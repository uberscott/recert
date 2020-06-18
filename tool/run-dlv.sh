#!/bin/bash

cd $(dirname $0)

export WATCH_NAMESPACE=recert
export SERVICE_ACCOUNT=operator
export POD_NAME=recert-operator
export NAMESPACE=recert
export OPERATOR_NAME=operator

dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec  ../go/bin/operator > ../out/dlv.log 2> ../out/dlv.log &

tail -f ../out/dlv.log 
