#!/bin/bash
FILE=/usr/bin/consul_exporter
if [ -f "$FILE" ]; then
    echo "$FILE exists."
else 
    curl -L -o consul-exporter.tar.gz https://github.com/prometheus/consul_exporter/releases/download/v0.8.0/consul_exporter-0.8.0.linux-amd64.tar.gz
    gunzip consul-exporter.tar.gz
    tar xf consul-exporter.tar
    rm consul-exporter.tar
    mv consul_exporter-0.8.0.linux-amd64/consul_exporter /usr/bin/consul_exporter
    chmod 755 /usr/bin/consul_exporter
    rm -rf consul-export*
    rm -rf consul_export*
fi

