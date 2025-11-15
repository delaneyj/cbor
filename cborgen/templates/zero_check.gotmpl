{{/*
Expression used for omitempty zero checks.

Inputs:
  .Receiver  - receiver identifier (e.g. "x")
  .Field     - Go field name (exported)
  .Kind      - one of:
               "string", "bool", "numeric",
               "time",
               "ptrOrInterface", "slice", "map".
*/}}
{{define "zeroCheck"}}
{{- if eq .Kind "string" -}}
{{ .Receiver }}.{{ .Field }} == ""
{{- else if eq .Kind "bool" -}}
!{{ .Receiver }}.{{ .Field }}
{{- else if eq .Kind "numeric" -}}
{{ .Receiver }}.{{ .Field }} == 0
{{- else if eq .Kind "time" -}}
{{ .Receiver }}.{{ .Field }}.IsZero()
{{- else if eq .Kind "ptrOrInterface" -}}
{{ .Receiver }}.{{ .Field }} == nil
{{- else if eq .Kind "slice" -}}
len({{ .Receiver }}.{{ .Field }}) == 0
{{- else if eq .Kind "map" -}}
len({{ .Receiver }}.{{ .Field }}) == 0
{{- end -}}
{{end}}
