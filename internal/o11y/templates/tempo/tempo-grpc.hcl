service {
  name = "tempo-grpc"
  address = "HOST"
  port = 4317
  tagged_addresses {
    lan = {
      address = "HOST"
      port = 4317
    }
  }
}