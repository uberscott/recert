#!/bin/bash

cd $(dirname $0)/..

diff ./tool/VERSION  .tool.version 2> /dev/null > /dev/null || (echo "tools are out of date." && echo "please run 'make tools' " && exit 1)

