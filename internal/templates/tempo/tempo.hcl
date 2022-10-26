service {
  name = "tempo"
  address = "HOST"
  port = 3200
  tagged_addresses {
    lan = {
      address = "HOST"
      port = 3200
    }
  }
}

