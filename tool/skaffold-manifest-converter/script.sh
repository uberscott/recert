#!/bin/bash

cp /dev/stdin .file.json

echo "{" > tmp.json
jq -r ".builds|keys[]" .file.json | while read key; do
  echo "$(jq .builds[$key].imageName .file.json | sed s#^.\*/#\"#g | sed -r 's/(^|-)([a-z])/\U\2/g' ):$(jq .builds[$key].tag .file.json)," >> tmp.json
done
echo "}" >> tmp.json

yq -y . < tmp.json

rm .file.json
