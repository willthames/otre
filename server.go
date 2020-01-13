package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	v1 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v1"
	v2 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v2"
	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/traces"
)

type key int

const (
	requestIDKey key = 0
)

// handleSpans handles the /api/v1/spans POST endpoint. It decodes the request
// body and normalizes it to a slice of types.Span instances. The Sink
// handles that slice. The Mirror, if configured, takes the request body
// verbatim and sends it to another host.
func (a *app) handleSpans(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Error("Error reading request body")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error reading request"))
	}

	contentType := r.Header.Get("Content-Type")

	var spans []*types.Span
	switch contentType {
	case "application/json":
		logrus.Info("Receiving data in json format")
		switch r.URL.Path {
		case "/api/v1/spans":
			spans, err = v1.DecodeJSON(bytes.NewReader(data))
		case "/api/v2/spans":
			spans, err = v2.DecodeJSON(bytes.NewReader(data))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid version"))
			return
		}
	case "application/x-thrift":
		logrus.Debug("Receiving data in thrift format")
		switch r.URL.Path {
		case "/api/v1/spans":
			spans, err = v1.DecodeThrift(bytes.NewReader(data))
		case "/api/v2/spans":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("thrift is not supported for v2 spans"))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid version"))
			return
		}
	default:
		logrus.WithField("contentType", contentType).Error("unknown content type")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unknown content type"))
		return
	}
	if err != nil {
		logrus.WithError(err).WithField("type", contentType).Error("error unmarshaling spans")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error unmarshaling span data"))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	for _, span := range spans {
		logrus.WithField("spanID", span.ID).Debug("Adding span to tracebuffer")
		a.traceBuffer.AddSpan(*span)
		logrus.WithField("spanID", span.ID).Debug("Finished adding span to tracebuffer")
	}
}

// ungzipWrap wraps a handleFunc and transparently ungzips the body of the
// request if it is gzipped
func ungzipWrap(hf func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var newBody io.ReadCloser
		isGzipped := r.Header.Get("Content-Encoding")
		if isGzipped == "gzip" {
			buf := bytes.Buffer{}
			if _, err := io.Copy(&buf, r.Body); err != nil {
				logrus.WithError(err).Error("error allocating buffer for ungzipping")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error allocating buffer for ungzipping"))
				return
			}
			var err error
			newBody, err = gzip.NewReader(&buf)
			if err != nil {
				logrus.WithError(err).Error("error ungzipping span data")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error ungzipping span data"))
				return
			}
			r.Body = newBody
		}
		hf(w, r)
	}
}

func (a *app) start() error {
	outputLines := [5]string{
		`       _`,
		`  ___ | |_ _ __ ___`,
		` / _ \| __| '__/ _ \`,
		`| (_) | |_| | |  __/`,
		` \___/ \__|_|  \___|`,
	}
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", ungzipWrap(a.handleSpans))
	mux.HandleFunc("/api/v2/spans", ungzipWrap(a.handleSpans))
	mux.HandleFunc("/", http.NotFoundHandler().ServeHTTP)

	a.server = &http.Server{
		Addr:     fmt.Sprintf(":%d", a.port),
		Handler:  logging(logger)(mux),
		ErrorLog: logger,
	}
	go a.server.ListenAndServe()
	for _, line := range outputLines {
		fmt.Println(line)
	}
	logrus.WithField("port", a.port).Info("Listening")
	ticker := time.NewTicker(a.flushAge)
	go a.scheduler(ticker)
	return nil
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func (a *app) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

func (a *app) scheduler(tick *time.Ticker) {
	for {
		select {
		case <-tick.C:
			a.processSpans()
		}
	}
}

func (a *app) writeTrace(trace *traces.Trace, sampleResult rules.SampleResult) error {
	body, err := trace.MarshalJSON()
	if err != nil {
		logrus.WithError(err).WithField("trace", trace).Error("Error converting trace to JSON")
		return err
	}
	if a.forwarder != nil {
		if err := a.forwarder.Send(payload{ContentType: "application/json", Body: body}); err != nil {
			logrus.WithError(err).Error("Error forwarding trace")
			logrus.WithField("body", body).Debug("Error forwarding trace body")
			return err
		}
		logrus.WithField("reason", sampleResult.Reason).WithField("trace", trace).Debug("accepting trace")
	} else {
		logrus.WithField("reason", sampleResult.Reason).WithField("trace", trace).Info("dry-run: would have accepted trace")
	}
	return nil
}

func (a *app) processSpans() {
	var decision bool
	var sampleResult rules.SampleResult
	var traceID traces.TraceID
	var trace *traces.Trace

	logrus.Debug("processSpans: RLocking tracebuffer")
	deletions := []traces.TraceID{}
	now := time.Now()
	a.traceBuffer.RLock()
	for traceID, trace = range a.traceBuffer.Traces {
		if trace.IsComplete() && trace.OlderThanRelative(a.flushAge, now) {
			decision, sampleResult = a.re.AcceptTrace(trace)
			if decision {
				trace.AddStringTag("SampleReason", sampleResult.Reason)
				trace.AddIntTag("SampleRate", sampleResult.SampleRate)
				err := a.writeTrace(trace, sampleResult)
				if err != nil {
					deletions = append(deletions, traceID)
				}
			} else {
				logrus.WithField("reason", sampleResult.Reason).WithField("trace", trace).Debug("dropping trace")
			}
		} else if trace.OlderThanRelative(a.abandonAge, now) {
			trace.AddStringTag("SampleReason", fmt.Sprintf("trace is older than abandonAge %dms", a.abandonAge))
			err := a.writeTrace(trace, sampleResult)
			if err != nil {
				deletions = append(deletions, traceID)
			}
		}
	}
	logrus.Debug("processSpans: RUnlocking tracebuffer")

	a.traceBuffer.RUnlock()
	logrus.Debug("processSpans: Locking tracebuffer")
	a.traceBuffer.Lock()
	for _, traceID = range deletions {
		delete(a.traceBuffer.Traces, traceID)
	}
	logrus.Debug("processSpans: Unlocking tracebuffer")
	a.traceBuffer.Unlock()
}

func main() {

	a := cliParse()
	level, err := logrus.ParseLevel(a.logLevel)
	if err != nil {
		logrus.WithField("logLevel", a.logLevel).Warn("Couldn't parse log level - defaulting to Info")
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(level)
	}
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if a.collectorURL != "" {
		logrus.WithField("collectorURL", a.collectorURL).Debug("Creating trace forwarder")
		a.forwarder, err = NewForwarder(a.collectorURL)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(1)
		}
		a.forwarder.Start()
		defer a.forwarder.Stop()
	} else {
		a.forwarder = nil
	}
	err = a.start()
	if err != nil {
		fmt.Printf("Error starting app: %v\n", err)
		os.Exit(1)
	}
	defer a.stop()
	waitForSignal()
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	defer close(ch)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)
	<-ch
}
