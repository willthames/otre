package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/spans"
	"github.com/willthames/otre/traces"
)

type key int

const (
	requestIDKey key = 0
)

var (
	incompleteTraces = promauto.NewCounter(prometheus.CounterOpts{
		Name: "otre_traces_incomplete_total",
		Help: "The total number of incomplete traces",
	})
	acceptedTraces = promauto.NewCounter(prometheus.CounterOpts{
		Name: "otre_traces_accepted_total",
		Help: "The total number of accepted traces",
	})
	rejectedTraces = promauto.NewCounter(prometheus.CounterOpts{
		Name: "otre_traces_rejected_total",
		Help: "The total number of rejected traces",
	})
	timedOutTraces = promauto.NewCounter(prometheus.CounterOpts{
		Name: "otre_traces_timed_out_total",
		Help: "The total number of traces unable to be sent before timing out",
	})
	tracesInBuffer = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "otre_traces_in_buffer",
		Help: "The number of traces currently in the buffer",
	})
	spansInBuffer = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "otre_spans_in_buffer",
		Help: "The number of spans currently in the buffer",
	})
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

	var result []*spans.Span
	switch contentType {
	case "application/json":
		logrus.Debug("Receiving data in json format")
		switch r.URL.Path {
		case "/api/v1/spans":
			err = json.Unmarshal(data, &result)
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid version"))
			return
		}
	case "application/x-thrift":
		logrus.Debug("Receiving data in thrift format")
		switch r.URL.Path {
		case "/api/v1/spans":
			result, err = spans.DecodeThrift(data)
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("thrift only unsupported for v1"))
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
	var tbm traces.TraceBufferMetrics
	for _, span := range result {
		logrus.WithField("spanID", span.ID).Debug("Adding span to tracebuffer")
		tbm = a.traceBuffer.AddSpan(*span)
		spansInBuffer.Add(float64(tbm.SpanDelta))
		tracesInBuffer.Add(float64(tbm.TraceDelta))
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
		Handler:  promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, logging(logger)(mux)),
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

func (a *app) writeTrace(trace *traces.Trace) error {
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
		logrus.WithField("trace", trace).Debug("accepting trace")
	} else {
		logrus.WithField("trace", trace).Info("dry-run: would have accepted trace")
	}
	return nil
}

func (a *app) processSpans() {
	var traceID traces.TraceID
	var trace *traces.Trace

	logrus.Debug("processSpans: RLocking tracebuffer")
	deletions := []traces.TraceID{}
	now := time.Now()
	a.traceBuffer.RLock()
	for traceID, trace = range a.traceBuffer.Traces {

		if trace.IsComplete() && trace.OlderThanRelative(a.flushAge, now) {
			if trace.SampleResult != nil {
				if trace.SampleDecision {
					err := a.writeTrace(trace)
					if err != nil {
						deletions = append(deletions, traceID)
						if strings.HasPrefix(trace.SampleResult.Reason, "trace is older than abandonAge") {
							incompleteTraces.Inc()
						} else {
							acceptedTraces.Inc()
						}
					} else if trace.OlderThanRelative(a.flushTimeout, now) {
						deletions = append(deletions, traceID)
						logrus.WithField("flushTimeout", a.flushTimeout).Warn("Couldn't write trace to collector within timeout")
						logrus.WithField("trace", trace).Debug("Timed out trace")
						timedOutTraces.Inc()
					}
				}
			} else {
				trace.SampleDecision, trace.SampleResult = a.re.AcceptSpans(trace.Spans())
				if trace.SampleDecision {
					trace.AddTag("SampleReason", trace.SampleResult.Reason)
					trace.AddTag("SampleRate", fmt.Sprintf("%d", trace.SampleResult.SampleRate))
					err := a.writeTrace(trace)
					if err != nil {
						deletions = append(deletions, traceID)
						acceptedTraces.Inc()
					}
				} else {
					logrus.WithField("trace", trace).Debug("rejecting trace through sampling")
					deletions = append(deletions, traceID)
					rejectedTraces.Inc()
				}
			}
		} else if trace.OlderThanRelative(a.abandonAge, now) {
			reason := fmt.Sprintf("trace is older than abandonAge %dms", a.abandonAge)
			trace.AddTag("SampleReason", reason)
			trace.SampleResult = &rules.SampleResult{SampleRate: 100, Reason: reason}
			trace.SampleDecision = true
			err := a.writeTrace(trace)
			if err != nil {
				deletions = append(deletions, traceID)
				incompleteTraces.Inc()
			}
		}
	}
	logrus.Debug("processSpans: RUnlocking tracebuffer")
	a.traceBuffer.RUnlock()
	var tbm traces.TraceBufferMetrics
	for _, traceID = range deletions {
		tbm = a.traceBuffer.DeleteTrace(traceID)
		spansInBuffer.Add(float64(tbm.SpanDelta))
		tracesInBuffer.Add(float64(tbm.TraceDelta))
	}
}

func main() {
	prometheus.Register(incompleteTraces)
	prometheus.Register(acceptedTraces)
	prometheus.Register(rejectedTraces)
	prometheus.Register(timedOutTraces)
	prometheus.Register(spansInBuffer)
	prometheus.Register(tracesInBuffer)
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

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", a.metricsPort), nil)
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
