#!/bin/sh
DOMAIN=$1
NAME=$(echo $DOMAIN | cut -d '.' -f 1)
mkdir -p certs
openssl req -subj '/CN=$DOMAIN' -new -newkey rsa:4096 -sha256 -days 365 -nodes -x509 -keyout certs/$NAME.key -out certs/$NAME.crt
