{{/* vim: set filetype=mustache: */}}
{{/* https://github.com/Masterminds/sprig/blob/bf29da0d74f74aeb5c3e3e7207eab76c28ac4049/functions.go#L182 */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "edb.name" -}}
{{- default .Values.image .Values.name -}}
{{- end -}}

{{/*
Number of standbys
*/}}
{{- define "edb.replicas" -}}
{{- default .Values.replication.standbys 2 | min 0 | add 1  -}}
{{- end -}}

{{/*
Return the proper Storage Class
*/}}
{{- define "edb.storageClass" -}}
    {{- if .Values.persistence.storageClass -}}
        {{- if (eq "-" .Values.persistence.storageClass) -}}
            {{- printf "storageClassName: \"\"" -}}
        {{- else }}
            {{- printf "storageClassName: %s" .Values.persistence.storageClass -}}
        {{- end -}}
    {{- end -}}
{{- end -}}

{{/*
Render a value which is containing a template.
Usage:
{{ include "edb.tplValue" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}
{{- define "edb.tplValue" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}

{{- define "imagePullSecret" }}
{{- if .Values.repo.password }}
{{- printf "{\"auths\": {\"%s\": {\"auth\": \"%s\"}}}" .Values.repo.registry (printf "%s:%s" .Values.repo.username .Values.repo.password | b64enc) | b64enc }}
{{- end }}
{{- end }}
