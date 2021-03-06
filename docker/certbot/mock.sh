#!/bin/bash

echo $@

set -e

DOMAIN=$1
EMAIL=$2
SECRET=$3

#certbot certonly --standalone -d $DOMAIN --email $EMAIL --non-interactive --agree-tos --dry-run


echo "writing mocks for $DOMAIN"
mkdir -p /etc/letsencrypt/live/$DOMAIN
echo "mock" > /etc/letsencrypt/live/$DOMAIN/fullchain.pem
echo "mock" > /etc/letsencrypt/live/$DOMAIN/privkey.pem
 

./update-kube-secrets.sh $SECRET-new
