package otre

accept = false {
  url := input[_].binaryAnnotations["http.url"]
  trace(url)
  endswith(url, "/ping")
  # msg := sprintf("URL ending /ping is a ping URL")
} else = true {
  url := input[_].binaryAnnotations["http.url"]
  trace(url)
  contains(url, "/api/newService")
  # msg := sprintf("URL /api/newService is for the new service with 100% sampling")
} else = true {
  status := to_number(input[_].binaryAnnotations["http.status_code"])
  status >= 500
  # msg := sprintf("Status code %v >= 500", [status])
} else = true {
   # msg := "Default sample rate"
   # change this to true to test with policy_test.rego
   percentChance(25)
} else = false {
  # msg := "Fallback rejection rule"
  true
}
