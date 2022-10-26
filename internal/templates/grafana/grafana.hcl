service {
  name = "grafana"
  address = "HOST"
  port = 3000
  check = {
    id = "grafana"
    http = "http://HOST:3000/login"
    method = "GET"
    disable_redirects = true
    interval = "20s"
    timeout = "1s"
  }
  tags = ["urlprefix-grafana.ROOTDOMAIN/"]
}