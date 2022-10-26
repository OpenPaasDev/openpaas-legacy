job "ig-bridge-demo" {

  datacenters = ["dc1"]

  # This group will have a task providing the ingress gateway automatically
  # created by Nomad. The ingress gateway is based on the Envoy proxy being
  # managed by the docker driver.
  group "ingress-group" {

    network {
      mode = "bridge"

      # This example will enable tcp traffic to access the uuid-api connect
      # native example service by going through the ingress gateway on port 8080.
      # The communication between the ingress gateway and the upstream service occurs
      # through the mTLS protected service mesh.
      port "api" {
        static = 8080
        to     = 8080
      }
    }

    service {
      name = "my-ingress-service"
      port = "8080"

      connect {
        gateway {

          # Consul gateway [envoy] proxy options.
          proxy {
            # The following options are automatically set by Nomad if not
            # explicitly configured when using bridge networking.
            #
            # envoy_gateway_no_default_bind = true
            # envoy_gateway_bind_addresses "uuid-api" {
            #   address = "0.0.0.0"
            #   port    = <associated listener.port>
            # }
            #
            # Additional options are documented at
            # https://www.nomadproject.io/docs/job-specification/gateway#proxy-parameters
          }

          # Consul Ingress Gateway Configuration Entry.
          ingress {
            # Nomad will automatically manage the Configuration Entry in Consul
            # given the parameters in the ingress block.
            #
            # Additional options are documented at
            # https://www.nomadproject.io/docs/job-specification/gateway#ingress-parameters
            listener {
              port     = 8080
              protocol = "tcp"
              service {
                name = "uuid-api"
              }
            }
          }
        }
      }
    }
  }

  # The UUID generator from the Connect-native demo is used as an example service.
  # The ingress gateway above makes access to the service possible over tcp port 8080.
  group "generator" {
    network {
      mode = "host"
      port "api" {}
    }

    service {
      name = "uuid-api"
      port = "${NOMAD_PORT_api}"

      connect {
        native = true
      }
    }

    task "generate" {
      driver = "docker"

      config {
        image        = "hashicorpnomad/uuid-api:v3"
        network_mode = "host"
      }

      env {
        BIND = "0.0.0.0"
        PORT = "${NOMAD_PORT_api}"
      }
    }
  }
}