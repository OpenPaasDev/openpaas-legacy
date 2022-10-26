storage "consul" {
  path    = "vault/"
  address =  "https://127.0.0.1:8501"
  // tls_ca_file = "/etc/vault.d/certs/consul-agent-ca.pem"
  tls_cert_file = "/etc/vault.d/certs/consul-agent-ca.pem"
  tls_key_file = "/etc/vault.d/certs/consul-agent-ca-key.pem"

  tls_skip_verify = true
  token = "{{VAULT_CONSUL_TOKEN}}"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  // tls_disable = true
  // tls_client_ca_file = "/etc/vault.d/certs/consul-agent-ca.pem"
  tls_cert_file = "/etc/vault.d/certs/tls.crt"
  tls_key_file = "/etc/vault.d/certs/tls.key"
}

disable_mlock = true

api_addr = "https://{{private_ip}}:8200"
cluster_addr = "https://{{private_ip}}:8201"
ui = true

telemetry {
  disable_hostname = true
  prometheus_retention_time = "12h"
}