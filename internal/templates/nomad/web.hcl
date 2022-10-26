job "healthcheck" {
  datacenters = ["{DATACENTRE}"]
  type = "service"

  group "healthcheck" {
    network {
      port "http" {
        to = 80
      }
    }

    service {
      name = "healthcheck"
      tags = ["urlprefix-/"]
      port = "http"
      check {
        name     = "alive"
        type     = "http"
        path     = "/"
        interval = "30s"
        timeout  = "2s"
      }
    }

    restart {
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }

    task "nginx" {
      driver = "docker"
      config {
        image = "nginx:latest"
        ports = ["http"]
      }
      resources {
        cpu        = 200
        memory     = 64
      }
    }
  }
}