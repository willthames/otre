package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/willthames/opentracing-processor/processor"
	"github.com/willthames/opentracing-processor/span"
	"github.com/willthames/otre/rules"
	"github.com/willthames/otre/traces"
)

type OtreApp struct {
	processor.App
	flushAge     time.Duration
	flushTimeout time.Duration
	abandonAge   time.Duration
	traceBuffer  *traces.TraceBuffer
	re           rules.RulesEngine
}

func NewOtreApp() *OtreApp {
	otreApp := new(OtreApp)
	outputLines := [5]string{
		`       _`,
		`  ___ | |_ _ __ ___`,
		` / _ \| __| '__/ _ \`,
		`| (_) | |_| | |  __/`,
		` \___/ \__|_|  \___|`,
	}
	otreApp.OutputLines = outputLines[:]
	otreApp.BaseCLI()
	flushAge := flag.Int("flush-age", 30000, "Interval in ms between trace flushes")
	flushTimeout := flag.Int("flush-timeout", 600000, "Drop traces older than timeout if not successfully forwarded to collector")
	abandonAge := flag.Int("abandon-age", 300000, "Age in ms after which incomplete trace is flushed")
	policyFile := flag.String("policy-file", "", "policy definition file")
	flag.Parse()
	otreApp.flushAge = time.Duration(*flushAge * 1E6)
	otreApp.flushTimeout = time.Duration(*flushTimeout * 1E6)
	otreApp.abandonAge = time.Duration(*abandonAge * 1E6)

	if *policyFile == "" {
		logrus.Fatal("--policy-file argument is mandatory")
	}
	policy, err := ioutil.ReadFile(*policyFile)
	if err != nil {
		panic(err)
	}
	otreApp.re = *rules.NewRulesEngine(string(policy))
	return otreApp
}

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

func (a *OtreApp) receiveSpan(span span.Span) {
	var tbm traces.TraceBufferMetrics
	logrus.WithField("spanID", span.ID).Debug("Adding span to tracebuffer")
	tbm = a.traceBuffer.AddSpan(span)
	spansInBuffer.Add(float64(tbm.SpanDelta))
	tracesInBuffer.Add(float64(tbm.TraceDelta))
	logrus.WithField("spanID", span.ID).Debug("Finished adding span to tracebuffer")
}

func (a *OtreApp) scheduler(tick *time.Ticker) {
	for {
		select {
		case <-tick.C:
			a.processSpans()
		}
	}
}

func (a *OtreApp) writeTrace(trace *traces.Trace) error {
	body, err := trace.MarshalJSON()
	if err != nil {
		logrus.WithError(err).WithField("trace", trace).Error("Error converting trace to JSON")
		return err
	}
	if a.Forwarder != nil {
		if err := a.Forwarder.Send(processor.Payload{ContentType: "application/json", Body: body}); err != nil {
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

func (a *OtreApp) processSpans() {
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
					err := trace.AddTag("SampleReason", trace.SampleResult.Reason)
					if err != nil {
						logrus.WithField("traceID", traceID).WithError(err).Warn("Couldn't add tag")
					}
					err = trace.AddTag("SampleRate", fmt.Sprintf("%d", trace.SampleResult.SampleRate))
					if err != nil {
						logrus.WithField("traceID", traceID).WithError(err).Warn("Couldn't add tag")
					}
					err = a.writeTrace(trace)
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
			err := trace.AddTag("SampleReason", reason)
			if err != nil {
				logrus.WithField("traceID", traceID).WithError(err).Warn("Couldn't add tag")
			}
			trace.SampleResult = &rules.SampleResult{SampleRate: 100, Reason: reason}
			trace.SampleDecision = true
			err = a.writeTrace(trace)
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

	a := NewOtreApp()
	ticker := time.NewTicker(a.flushAge)
	go a.scheduler(ticker)
	a.Serve()
}
