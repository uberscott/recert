#!/bin/bash

set -e

cd $(dirname $0)/..

pwd 
echo "building tool"


for TOOL in $(ls ./tool)
do

  if [ -d ./tool/$TOOL ]
  then
    echo 
    echo "Building $TOOL"
    docker build . -f tool/$TOOL/Dockerfile --tag tool_$TOOL
  fi

done


# increment .tool.version if there are any updates to tools
cp tool/VERSION .tool.version


# write pre-commit hooks

cp tool/pre-commit ./.git/hooks/

