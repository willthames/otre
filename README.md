Opentracing policy proxy
------------------------

The opentracing policy proxy is designed as a proxy that provides
a means of deciding which traces to forward on to the next hop.

By providing an open policy agent ruleset against which to check
rules, combined with a store of traces and accept/reject
results, entire traces can be dropped.

In general, we want to be able to set policies such:
* Sample 25% of /api endpoints. Unless it's the
  /api/newfeature endpoint, in which case sample 100%
* For services labelled with the canary tag, take the
  canary percentage and ensure that at least the standard
  sample rate is processed. This means for a standard
  25% sample rate, with a 1% canary, sample 100% of traffic
  but for a 50% canary sample 50% of traffic.
* Sample all traces that take longer than X seconds
* Drop all /api/ping traces
* Drop all traces with a reject tag

By default all traces are accepted. A trace id then might
be rejected because of the sample rate of the endpoint
at which point the trace id is tagged with reject. However,
1 minute later, traces with that trace id turn up with durations
longer than 60 seconds, at which point the trace id gets
removed from the reject store.



Minimum viable feature set
==========================

* in memory store of traces on a single host configurable
  by max store size, max time traces stay in store before
  flush (should be longer than any trace duration policy)
* /api/v1/spans zipkin endpoint forwarding to a zipkin
  proxy
* Debugging rejection reasons

Longer term feature set
=======================

* distributed store of traces and reject states
* /api/v2/spans, jaeger endpoint, opentracing, opentelemetry
