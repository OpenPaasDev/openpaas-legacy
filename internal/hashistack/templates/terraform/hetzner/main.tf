terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "1.34.3"
    }
  }
}

# Configure the Hetzner Cloud Provider
provider "hcloud" {
  token = var.hcloud_token
}

locals {
  # Common tags to be assigned to all resources

  consul_group = var.separate_consul_servers ? "consul" : "nomad-server"
  groups = {
    consul = {
      count = var.separate_consul_servers ? var.server_count : 0 * var.server_count, 
      subnet = "0", group = 0, server_type = var.server_instance_type
    }
    nomad-server = {
      count = var.server_count, subnet = "1", group = 0, server_type = var.server_instance_type
    },
    vault = {
      count = var.vault_count, subnet = "2", group = 0, server_type = var.server_instance_type
    },
    client = {
      count = var.client_count, subnet = "3", group = 1, server_type = var.client_instance_type
    },
    observability = {
      count = var.multi_instance_observability ? 4 : 1, subnet = "4", group = 0, server_type = var.observability_instance_type
    }
  }

  servers = flatten([
    for name, value in local.groups : [
      for i in range(value.count) : {
        group_name = name,
        private_ip = "10.0.${value.subnet}.${i + 2}",
        name       = "${var.base_server_name}-${name}-${i + 1}",
        group      = value.group
        index = i
        server_type = value.server_type
      }
    ]
  ])

  placement_groups = 2
}

resource "hcloud_network" "private_network" {
  name     = var.network_name
  ip_range = "10.0.0.0/16"
}

resource "hcloud_network_subnet" "network_subnet" {
  for_each     = local.groups
  network_id   = hcloud_network.private_network.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = "10.0.${each.value.subnet}.0/24"
}

resource "hcloud_placement_group" "placement_group" {
  count = local.placement_groups
  name  = "server_placement_spread_group-${count.index}"
  type  = "spread"
}


resource "hcloud_firewall" "network_firewall" {
  name = var.firewall_name
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "1-10000"
    source_ips = var.allow_ips
  }

  rule {
    direction = "in"
    protocol  = "icmp"
    source_ips = [
      "0.0.0.0/0",
      "::/0"
    ]
  }
}

resource "hcloud_server" "server_node" {
  for_each           = { for entry in local.servers : "${entry.name}" => entry }
  name               = each.value.name
  image              = "ubuntu-22.04"
  server_type        = each.value.server_type
  location           = var.location
  placement_group_id = hcloud_placement_group.placement_group[each.value.group].id
  firewall_ids       = [hcloud_firewall.network_firewall.id]

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }
  depends_on = [
    hcloud_network_subnet.network_subnet["consul"],
    hcloud_network_subnet.network_subnet["nomad-server"],
    hcloud_network_subnet.network_subnet["vault"],
    hcloud_network_subnet.network_subnet["client"],
    hcloud_network_subnet.network_subnet["observability"],
  ]

  labels = {
    "group" = each.value.group_name
  }

  ssh_keys = var.ssh_keys
}

resource "hcloud_server_network" "network_binding" {
  for_each   = { for entry in local.servers : "${entry.name}" => entry }
  server_id  = hcloud_server.server_node[each.value.name].id
  network_id = hcloud_network.private_network.id
  ip         = each.value.private_ip
}


resource "hcloud_volume" "consul" {
  count = var.server_count
  location = var.location
  name     = "consul-${count.index}"
  size     = var.consul_volume_size
  format   = "ext4"
  depends_on = [
    hcloud_server.server_node 
  ]
}

resource "hcloud_volume" "client_volumes" {
  for_each = { for entry in var.client_volumes : "${entry.name}" => entry.size }
  location = var.location
  name     = "${each.key}"
  size     = each.value
  format   = "ext4"
  depends_on = [
    hcloud_server.server_node 
  ]
}

resource "hcloud_volume_attachment" "client_volumes" {
  for_each = { for index, entry in var.client_volumes : entry.name => entry.client }
  volume_id = hcloud_volume.client_volumes[each.key].id
  server_id = hcloud_server.server_node[each.value].id
  automount = true

  depends_on = [
    hcloud_volume.client_volumes 
  ]
}

resource "hcloud_volume_attachment" "consul" {
  for_each = {for key, val in local.servers: val.index => val.name if val.group_name == local.consul_group}
  volume_id = hcloud_volume.consul[each.key].id
  server_id = hcloud_server.server_node[each.value].id
  automount = true

  depends_on = [
    hcloud_volume.consul 
  ]
}

resource "hcloud_load_balancer" "lb1" {
  name               = "lb1"
  load_balancer_type = var.load_balancer_type
  # network_zone       =  hcloud_network_subnet.network_subnet["consul"].network_zone
  location           = var.location
  depends_on = [
    hcloud_server.server_node,
    hcloud_server_network.network_binding,
    hcloud_network_subnet.network_subnet["client"],
    hcloud_network_subnet.network_subnet["consul"],
  ]
}

resource "hcloud_load_balancer_network" "srvnetwork" {
  load_balancer_id = hcloud_load_balancer.lb1.id
  network_id       = hcloud_network.private_network.id
  ip               = "10.0.0.7" # max 5 consul servers, so 10.0.0.7 is free
  depends_on = [
    hcloud_network.private_network
  ]
}

resource "hcloud_load_balancer_service" "load_balancer_service" {
    load_balancer_id = hcloud_load_balancer.lb1.id
    protocol         = "https"
    destination_port = 80
    http {
      certificates = var.ssl_certificate_ids 
    }
}


# this is unfortunately necessary, because no amount of `depends_on` on the load_balancer_target will ensure
# the nodes and networks are ready for load_balancer target attachment, other than waiting
resource "time_sleep" "wait" {
  create_duration = "30s"
  depends_on = [
    hcloud_server.server_node,
    hcloud_server_network.network_binding,
    hcloud_network_subnet.network_subnet["client"],
  ]
}


resource "hcloud_load_balancer_target" "load_balancer_target" {
  for_each = {for key, val in local.servers: val.index => val.name if val.group_name == "client"}
  type             = "server"
  load_balancer_id = hcloud_load_balancer.lb1.id
  server_id        = hcloud_server.server_node[each.value].id
  use_private_ip = true
  depends_on = [time_sleep.wait]
}


output "consul_servers" {
  value = flatten([
    for index, node in hcloud_server.server_node : [
      for server in local.servers :
      {host = "${node.ipv4_address}", 
        host_name = "${node.name}", 
        private_ip = "${server.private_ip}",
        server_id= node.id
      } if server.name == node.name
    ] if node.labels["group"] == "consul"
  ])
}

output "nomad_servers" {
  value = flatten([
    for index, node in hcloud_server.server_node : [
      for server in local.servers :
      {host = "${node.ipv4_address}", 
        host_name = "${node.name}", 
        private_ip = "${server.private_ip}",
        server_id= node.id
      } if server.name == node.name
    ] if node.labels["group"] == "nomad-server"
  ])
}

output "vault_servers" {
  value = flatten([
    for index, node in hcloud_server.server_node : [
      for server in local.servers :
      {host = "${node.ipv4_address}", 
        host_name = "${node.name}", 
        private_ip = "${server.private_ip}",
        server_id= node.id
      } if server.name == node.name
    ] if node.labels["group"] == "vault"
  ])
}

output "client_servers" {
  value = flatten([
    for index, node in hcloud_server.server_node : [
      for server in local.servers :
      {host = "${node.ipv4_address}", 
        host_name = "${node.name}", 
        private_ip = "${server.private_ip}",
        server_id= node.id
      } if server.name == node.name
    ] if node.labels["group"] == "client"
  ])
}

output "o11y_servers" {
  value = flatten([
    for index, node in hcloud_server.server_node : [
      for server in local.servers :
      {host = "${node.ipv4_address}", 
        host_name = "${node.name}", 
        private_ip = "${server.private_ip}",
        server_id = node.id
      } if server.name == node.name
     ] if node.labels["group"] == "observability"
  ])
}

output "consul_volumes" {
  value = flatten([
    for index, attachment in hcloud_volume_attachment.consul : [
      {mount = "/mnt/HC_Volume_${attachment.volume_id}", 
      path = "/opt/consul",
      name = "",
      server_id = attachment.server_id,
      is_nomad = false
      }
    ] 
  ])
}

output "client_volumes" {
 value = flatten([
    for index, attachment in hcloud_volume_attachment.client_volumes : [
      for vol in var.client_volumes :
      {mount = "/mnt/HC_Volume_${attachment.volume_id}", 
      path = vol.path,
      name = vol.name,
      is_nomad = true,
      server_id = attachment.server_id} if hcloud_volume.client_volumes[index].name == vol.name
    ] 
  ])
}

