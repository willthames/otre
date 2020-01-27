package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/traces"
)

type app struct {
	port         int
	metricsPort  int
	server       *http.Server
	flushAge     time.Duration
	flushTimeout time.Duration
	abandonAge   time.Duration
	collectorURL string
	traceBuffer  *traces.TraceBuffer
	re           rules.RulesEngine
	forwarder    *Forwarder
	logLevel     string
}

func cliParse() *app {
	port := flag.Int("port", 8080, "server port")
	metricsPort := flag.Int("metrics-port", 10010, "prometheus /metrics port")
	flushAge := flag.Int("flush-age", 30000, "Interval in ms between trace flushes")
	flushTimeout := flag.Int("flush-timeout", 600000, "Drop traces older than timeout if not successfully forwarded to collector")
	abandonAge := flag.Int("abandon-age", 300000, "Age in ms after which incomplete trace is flushed")
	collectorURL := flag.String("collector-url", "", "Host to forward traces. Not setting this will work as dry run")
	policyFile := flag.String("policy-file", "", "policy definition file")
	logLevel := flag.String("log-level", "Info", "log level")

	flag.Parse()

	if *policyFile == "" {
		logrus.Fatal("--policy-file argument is mandatory")
	}
	policy, err := ioutil.ReadFile(*policyFile)
	if err != nil {
		panic(err)
	}
	a := &app{
		port:         *port,
		metricsPort:  *metricsPort,
		flushAge:     time.Duration(int64(*flushAge * 1E6)),
		abandonAge:   time.Duration(int64(*abandonAge * 1E6)),
		flushTimeout: time.Duration(int64(*flushTimeout * 1E6)),
		collectorURL: *collectorURL,
		logLevel:     *logLevel,
		traceBuffer:  traces.NewTraceBuffer(),
		re:           *rules.NewRulesEngine(string(policy)),
	}
	return a
}
