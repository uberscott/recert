#!/bin/bash

cd $(dirname $0)/..

for CRD in $(ls go/src/operator/deploy/crds/*_crd.yaml)
do
  echo "{{- if .Values.installCRDs -}}" > "helm/operator/chart/templates/$(basename $CRD)"
  cat $CRD >> "helm/operator/chart/templates/$(basename $CRD)"
  echo "{{- end -}}" >> "helm/operator/chart/templates/$(basename $CRD)"
done

