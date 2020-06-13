#!/bin/bash

set -e
set -x

rm /etc/letsencrypt/live/README 2>/dev/null || true

CERT=$1

OUT="/ssl2"

mkdir $OUT || true
cp -r /ssl/* $OUT

for dir in /etc/letsencrypt/live/*
do

  FULLCHAIN="$(realpath $dir/fullchain.pem)"
  PRIVKEY="$(realpath $dir/privkey.pem)"

  cp $FULLCHAIN $OUT/$(basename $dir).crt
  cp $PRIVKEY   $OUT/$(basename $dir).key

done

kubectl apply secret generic ssl --from-file=/ssl2/

kubectl patch cert $CERT -f updated.yaml

echo "DONE"
