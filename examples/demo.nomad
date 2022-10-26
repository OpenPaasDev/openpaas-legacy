job "demo" {

  datacenters = ["hetzner"]
  type = "service"

  update {
    max_parallel = 1
    min_healthy_time = "10s"
    healthy_deadline = "3m"
    progress_deadline = "10m"
    auto_revert = false
    canary = 0
  }

  migrate {
    max_parallel = 1
    health_check = "checks"
    min_healthy_time = "10s"
    healthy_deadline = "5m"
  }
  group "app" {

    count = 2

    network {
      mode = "bridge"
      port "http" {
        to = 8080
      }

      port "prometheus" {
        to = 8090
      }
    }

    service {
      name = "demo-app-http"
      tags = ["public"]
      port = "http"
      connect {
        sidecar_service {}
      }

      check {
        type     = "http"
        port     = "http"
        path     = "/_healthz"
        interval = "5s"
        timeout  = "2s"
        // header {
        //   Authorization = ["Basic ZWxhc3RpYzpjaGFuZ2VtZQ=="]
        // }
      }
    }

    service {
      name = "demo-app-prometheus"
      tags = ["prometheus"]
      port = "prometheus"
    }

    restart {
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }

    task "demo-app" {
      driver = "docker"

      vault {
        policies = ["backend"]

        change_mode   = "signal"
        change_signal = "SIGUSR1"
      }

      config {
        #image = "wfaler/demo-app:v16"
        image = "ghcr.io/chaordic-io/demo-app:v1"
        ports = ["http", "prometheus"]
        auth {
          username = "wfaler"
          password = "github token"
          server_address = "ghcr.io"
        }
      }
      env {
        PORT = "8080"
        PROMETHEUS_PORT = "8090"
      }

// vault policy write backend access.hcl
//consul kv put redis/config/connections 5
// vault kv put secret/geo-api-key foo=world
      template {
        data = <<EOH
# Lines starting with a # are ignored
{{- range service "tempo-grpc" }}
TEMPO_ENDPOINT="{{ .Address }}:{{ .Port }}"
{{- end }}
CONNECTIONS="{{key "redis/config/connections"}}"
API_KEY="{{with secret "secret/data/geo-api-key"}}{{.Data.data.foo}}{{end}}"
      EOH


        env         = true
        destination = "/app/env"
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
      }
    }
  }
}
