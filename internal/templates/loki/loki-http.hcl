service {
  name = "loki-http"
  address = "HOST"
  port = 3100
  tagged_addresses {
    lan = {
      address = "HOST"
      port = 3100
    }
  }
}