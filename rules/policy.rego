package otre

default url = ""
default status = 0

ping[span] {
  span := input[_]
  annotation := span.binaryAnnotations[_]
  annotation.key == "http.url"
  url :=  annotation.value
  endswith(url, "/ping")
}

api_new_service[span] {
  span := input[_]
  annotation := span.binaryAnnotations[_]
  annotation.key == "http.url"
  url :=  annotation.value
  contains(url, "/api/newService")
}

error_response[span] {
  span := input[_]
  annotation := span.binaryAnnotations[_]
  annotation.key == "http.status_code"
  status := to_number(annotation.value)
  status >= 500
}

response = {"sampleRate": 0, "reason": msg} {
  some span
  ping[span]
  msg := "URL ending /ping is a ping URL"
} else = {"sampleRate": 100, "reason": msg} {
  some span
  api_new_service[span]
  msg := "URL is for the new service with 100% sampling"
} else = {"sampleRate": 100, "reason": msg} {
  some span
  error_response[span]
  msg := "Status code >= 500"
}  else = {"sampleRate": 25, "reason": "fallback sample rate"} {
  true
}
