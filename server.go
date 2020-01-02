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
	"syscall"
	"time"

	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
	v1 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v1"
	v2 "github.com/honeycombio/honeycomb-opentracing-proxy/types/v2"
	"github.com/Sirupsen/logrus"
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
		logrus.WithError(err).Info("Error reading request body")
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
		logrus.Info("Receiving data in thrift format")
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
		logrus.WithField("contentType", contentType).Info("unknown content type")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("unknown content type"))
		return
	}
	if err != nil {
		logrus.WithError(err).WithField("type", contentType).Info("error unmarshaling spans")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("error unmarshaling span data"))
		return
	}
	var opa_data []byte
	for _, span := range spans {
		opa_data, err = json.Marshal(span)
		if err != nil {
			logrus.WithError(err).WithField("span", span).Info("error marshaling span")
		} else {
			logrus.WithField("span", string(opa_data)).Info("Accepted span")
		}
	}

	w.WriteHeader(http.StatusAccepted)
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
				logrus.WithError(err).Info("error allocating buffer for ungzipping")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error allocating buffer for ungzipping"))
				return
			}
			var err error
			newBody, err = gzip.NewReader(&buf)
			if err != nil {
				logrus.WithError(err).Info("error ungzipping span data")
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
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", ungzipWrap(a.handleSpans))
	mux.HandleFunc("/api/v2/spans", ungzipWrap(a.handleSpans))
	mux.HandleFunc("/", http.NotFoundHandler().ServeHTTP)

	a.server = &http.Server{
		Addr:     ":" + a.Port,
		Handler:  logging(logger)(mux),
		ErrorLog: logger,
	}
	go a.server.ListenAndServe()
	logrus.WithField("port", a.Port).Info("Listening")
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

func main() {
	a := cliParse()
	err := a.start()
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
