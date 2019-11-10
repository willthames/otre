package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/honeycombio/honeycomb-opentracing-proxy/types"
)

type App struct {
	Port   string
	server *http.Server
}

// handleSpans handles the /api/v1/spans POST endpoint. It decodes the request
// body and normalizes it to a slice of types.Span instances. The Sink
// handles that slice. The Mirror, if configured, takes the request body
// verbatim and sends it to another host.
func (a *App) handleSpans(w http.ResponseWriter, r *http.Request) {
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
		spans, err = types.DecodeJSON(bytes.NewReader(data))
	case "application/x-thrift":
		logrus.Info("Receiving data in thrift format")
		spans, err = types.DecodeThrift(bytes.NewReader(data))
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

	for _, span := range spans {
		logrus.WithField("span", span).Info("Accepted span")
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

func (a *App) start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/spans", ungzipWrap(a.handleSpans))

	a.server = &http.Server{
		Addr:    a.Port,
		Handler: mux,
	}
	go a.server.ListenAndServe()
	logrus.WithField("port", a.Port).Info("Listening")
	return nil
}

func (a *App) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return a.server.Shutdown(ctx)
}

func main() {
	port := flag.Int("port", 8080, "server port")
	flag.Parse()
	a := &App{
		Port: strconv.Itoa(*port),
	}
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
