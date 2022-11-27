service {
  name = "consul-ingress"
  address = "HOST"
  port = 8500
  check = {
    id = "consul-ingress"
    http = "http://HOST:8500/ui/dc1/services"
    method = "GET"
    disable_redirects = true
    interval = "20s"
    timeout = "1s"
  }
  tags = ["urlprefix-consul.ROOTDOMAIN/ auth=consul-auth"]
}