#!/bin/bash
FILE=/usr/local/bin/fabio
if [ -f "$FILE" ]; then
    echo "$FILE exists."
else 
    curl -L -o fabio https://github.com/fabiolb/fabio/releases/download/v1.6.1/fabio-1.6.1-linux_amd64
    mv fabio /usr/local/bin/fabio
    chmod 755 /usr/local/bin/fabio
fi