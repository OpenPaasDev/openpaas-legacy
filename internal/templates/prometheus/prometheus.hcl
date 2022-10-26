service {
  name = "prometheus"
  address = "HOST"
  port = 9090
  tagged_addresses {
    lan = {
      address = "HOST"
      port = 9090
    }
  }
}