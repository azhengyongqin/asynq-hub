{{/*
Expand the name of the chart.
*/}}
{{- define "asynqhub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "asynqhub.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "asynqhub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "asynqhub.labels" -}}
helm.sh/chart: {{ include "asynqhub.chart" . }}
{{ include "asynqhub.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "asynqhub.selectorLabels" -}}
app.kubernetes.io/name: {{ include "asynqhub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "asynqhub.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "asynqhub.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Backend full name
*/}}
{{- define "asynqhub.backend.fullname" -}}
{{- printf "%s-backend" (include "asynqhub.fullname" .) }}
{{- end }}

{{/*
Worker full name
*/}}
{{- define "asynqhub.worker.fullname" -}}
{{- printf "%s-worker" (include "asynqhub.fullname" .) }}
{{- end }}
