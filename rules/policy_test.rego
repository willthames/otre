package otre

import data.otre

trace_normal = [
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep","id":"b73350fb01c09f2d","parentId":"b007995ba1ff131e","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"flask","sleep":5},"timestamp":"2019-12-28T03:39:42.812652127Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"docker-debug.sleep","id":"b007995ba1ff131e","parentId":"7800b113b233ee63","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"Flask","http.method":"GET","http.url":"http://127.0.0.1:5000/sleep/5","span.kind":"server"},"timestamp":"2019-12-28T03:39:42.812669004Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep/5","id":"d9fecab3a39f9a73","serviceName":"nginx-ingress","durationMs":5013.407,"binaryAnnotations":{"component":"nginx","http.host":"localhost:8080","http.method":"GET","http.status_code":200,"http.status_line":"200 OK","http.url":"http://localhost:8080/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"nginx-ingress","nginx.worker_pid":7,"peer.address":"127.0.0.1:58868","request_id":"aaa0fb401020734fc6fd22dbd33abc96","upstream.address":"10.105.217.58:80","version":0.92},"timestamp":"2019-12-28T03:39:35.926Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep/5","id":"7800b113b233ee63","parentId":"d9fecab3a39f9a73","serviceName":"docker-debug","durationMs":5018.656,"binaryAnnotations":{"component":"nginx","http.host":"docker-debug.opa-tracing.svc.cluster.local","http.method":"GET","http.status_code":200,"http.status_line":"200 OK","http.url":"http://docker-debug.opa-tracing.svc.cluster.local/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"docker-debug","nginx.worker_pid":7,"peer.address":"10.1.3.57:45518","request_id":"f4c81620a0468a02c808833902fb85ff","upstream.address":"127.0.0.1:5000","version":""},"timestamp":"2019-12-28T03:39:35.929Z"}
]

trace_5xx = [
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep","id":"b73350fb01c09f2d","parentId":"b007995ba1ff131e","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"flask","sleep":5},"timestamp":"2019-12-28T03:39:42.812652127Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"docker-debug.sleep","id":"b007995ba1ff131e","parentId":"7800b113b233ee63","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"Flask","http.method":"GET","http.url":"http://127.0.0.1:5000/sleep/5","span.kind":"server"},"timestamp":"2019-12-28T03:39:42.812669004Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep/5","id":"d9fecab3a39f9a73","serviceName":"nginx-ingress","durationMs":5013.407,"binaryAnnotations":{"component":"nginx","http.host":"localhost:8080","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://localhost:8080/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"nginx-ingress","nginx.worker_pid":7,"peer.address":"127.0.0.1:58868","request_id":"aaa0fb401020734fc6fd22dbd33abc96","upstream.address":"10.105.217.58:80","version":0.92},"timestamp":"2019-12-28T03:39:35.926Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep/5","id":"7800b113b233ee63","parentId":"d9fecab3a39f9a73","serviceName":"docker-debug","durationMs":5018.656,"binaryAnnotations":{"component":"nginx","http.host":"docker-debug.opa-tracing.svc.cluster.local","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://docker-debug.opa-tracing.svc.cluster.local/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"docker-debug","nginx.worker_pid":7,"peer.address":"10.1.3.57:45518","request_id":"f4c81620a0468a02c808833902fb85ff","upstream.address":"127.0.0.1:5000","version":""},"timestamp":"2019-12-28T03:39:35.929Z"}
]

trace_api_newservice = [
  {"traceId":"17dc8c8a2c5f3ad5","name":"/sleep","id":"b73350fb01c09f2d","parentId":"b007995ba1ff131e","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"flask","sleep":5},"timestamp":"2019-12-28T03:39:42.812652127Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"docker-debug.sleep","id":"b007995ba1ff131e","parentId":"7800b113b233ee63","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"Flask","http.method":"GET","http.url":"http://127.0.0.1:5000/sleep/5","span.kind":"server"},"timestamp":"2019-12-28T03:39:42.812669004Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/api/newService/sleep/5","id":"d9fecab3a39f9a73","serviceName":"nginx-ingress","durationMs":5013.407,"binaryAnnotations":{"component":"nginx","http.host":"localhost:8080","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://localhost:8080/api/newService/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"nginx-ingress","nginx.worker_pid":7,"peer.address":"127.0.0.1:58868","request_id":"aaa0fb401020734fc6fd22dbd33abc96","upstream.address":"10.105.217.58:80","version":0.92},"timestamp":"2019-12-28T03:39:35.926Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/api/newService/sleep/5","id":"7800b113b233ee63","parentId":"d9fecab3a39f9a73","serviceName":"docker-debug","durationMs":5018.656,"binaryAnnotations":{"component":"nginx","http.host":"docker-debug.opa-tracing.svc.cluster.local","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://docker-debug.opa-tracing.svc.cluster.local/api/newService/sleep/5","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"docker-debug","nginx.worker_pid":7,"peer.address":"10.1.3.57:45518","request_id":"f4c81620a0468a02c808833902fb85ff","upstream.address":"127.0.0.1:5000","version":""},"timestamp":"2019-12-28T03:39:35.929Z"}
]
trace_ping = [
  {"traceId":"17dc8c8a2c5f3ad5","name":"/ping","id":"b73350fb01c09f2d","parentId":"b007995ba1ff131e","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"flask","sleep":5},"timestamp":"2019-12-28T03:39:42.812652127Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"docker-debug.ping","id":"b007995ba1ff131e","parentId":"7800b113b233ee63","serviceName":"docker-debug","hostIPv4":"10.1.3.71","binaryAnnotations":{"component":"Flask","http.method":"GET","http.url":"http://127.0.0.1:5000/ping","span.kind":"server"},"timestamp":"2019-12-28T03:39:42.812669004Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/ping","id":"d9fecab3a39f9a73","serviceName":"nginx-ingress","durationMs":5013.407,"binaryAnnotations":{"component":"nginx","http.host":"localhost:8080","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://localhost:8080/ping","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"nginx-ingress","nginx.worker_pid":7,"peer.address":"127.0.0.1:58868","request_id":"aaa0fb401020734fc6fd22dbd33abc96","upstream.address":"10.105.217.58:80","version":0.92},"timestamp":"2019-12-28T03:39:35.926Z"},
  {"traceId":"17dc8c8a2c5f3ad5","name":"/ping","id":"7800b113b233ee63","parentId":"d9fecab3a39f9a73","serviceName":"docker-debug","durationMs":5018.656,"binaryAnnotations":{"component":"nginx","http.host":"docker-debug.opa-tracing.svc.cluster.local","http.method":"GET","http.status_code":500,"http.status_line":"200 OK","http.url":"http://docker-debug.opa-tracing.svc.cluster.local/ping","http_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:71.0) Gecko/20100101 Firefox/71.0","lc":"docker-debug","nginx.worker_pid":7,"peer.address":"10.1.3.57:45518","request_id":"f4c81620a0468a02c808833902fb85ff","upstream.address":"127.0.0.1:5000","version":""},"timestamp":"2019-12-28T03:39:35.929Z"}
]

test_accept_with_5xx_error {
    otre.accept with input as trace_5xx
}

test_accept_with_api_newservice {
    otre.accept with input as trace_api_newservice
}

test_reject_with_ping {
    not otre.accept with input as trace_ping
}