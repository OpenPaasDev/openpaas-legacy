#!/bin/bash

rm prom*

wget https://github.com/grafana/loki/releases/download/v2.5.0/promtail-linux-amd64.zip
unzip promtail-linux-amd64.zip
mv promtail-linux-amd64 /usr/local/bin/promtail
chmod 755 /usr/local/bin/promtail

