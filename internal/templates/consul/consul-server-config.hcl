datacenter = "dc1"
data_dir = "/opt/consul"
encrypt = "{{ CONSUL_GOSSIP_KEY }}"
verify_incoming = true
verify_outgoing = true
verify_server_hostname = true

client_addr = "0.0.0.0"
server = true
bootstrap_expect = EXPECTS_NO

bind_addr = "{{ private_ip }}"

ca_file = "/etc/consul.d/certs/consul-agent-ca.pem"
cert_file = "/etc/consul.d/certs/dc1-server-consul-0.pem"
key_file = "/etc/consul.d/certs/dc1-server-consul-0-key.pem"

auto_encrypt {
  allow_tls = true
}


connect {
  enabled = true
}

retry_join = [join_servers]


ui_config {
  enabled = true
}

acl = {
  enabled = true
  default_policy = "deny"
  enable_token_persistence = true
}

ports {
  http = 8500
  https = 8501
}

telemetry {
  disable_hostname = true
  prometheus_retention_time = "12h"
}