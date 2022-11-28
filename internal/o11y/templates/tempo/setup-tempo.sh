#!/bin/bash

rm tempo*
wget https://github.com/grafana/tempo/releases/download/v1.5.0/tempo_1.5.0_linux_amd64.tar.gz
gunzip tempo_1.5.0_linux_amd64.tar.gz
tar xvf tempo_1.5.0_linux_amd64.tar
mv tempo /usr/local/bin/tempo
chmod 755 /usr/local/bin/tempo

