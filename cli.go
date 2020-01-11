package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/traces"
)

type app struct {
	port          int
	server        *http.Server
	flushAge      time.Duration
	abandonAge    time.Duration
	collectorURL  string
	traceBuffer   traces.TraceBuffer
	re            rules.RulesEngine
	forwarder     *Forwarder
	logLevel      string
}

func cliParse() *app {
	port := flag.Int("port", 8080, "server port")
	flushAge := flag.Int("flush-age", 30000, "Interval in ms between trace flushes")
	abandonAge := flag.Int("abandon-age", 300000, "Age in ms after which incomplete trace is flushed")
	collectorURL := flag.String("collector-url", "", "Host to forward traces. Not setting this will work as dry run")
	policyFile := flag.String("policy-file", "", "policy definition file")
	policyFile := flag.String("log-level", "Info", "log level")

	policy, err := ioutil.ReadFile(*policyFile)
	if err != nil {
		panic(err)
	}
	flag.Parse()
	a := &app{
		port:          *port,
		flushAge:      time.Duration(int64(*flushAge * 1E6)),
		abandonAge:    time.Duration(int64(*abandonAge * 1E6)),
		collectorURL:  *collectorURL,
		logLevel:      *logLevel,
		traceBuffer:   *new(traces.TraceBuffer),
		re:            *rules.NewRulesEngine(string(policy)),
	}
	return a
}
