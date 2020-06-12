#!/bin/bash

#Set the version

cd $(dirname $0)/..

VERSION=$1

echo "setting the project version to $VERSION"

find . -name .edb-version.sh -exec {} $VERSION \;

echo $VERSION > VERSION
