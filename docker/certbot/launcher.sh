#!/bin/bash

COMMAND=$1

if [ "$COMMAND" = "create" ]; then
  shift
  ./create.sh $@
  exit $?
fi 

echo "cannot process command: $COMMAND"
