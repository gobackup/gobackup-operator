{{/*
Expand the name of the chart.
*/}}
{{- define "gobackup-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "gobackup-operator.fullname" -}}
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
{{- define "gobackup-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gobackup-operator.labels" -}}
helm.sh/chart: {{ include "gobackup-operator.chart" . }}
{{ include "gobackup-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service | default "Helm" }}
app.kubernetes.io/part-of: gobackup-operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gobackup-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gobackup-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "gobackup-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gobackup-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "gobackup-operator.image" -}}
{{- $tag := "latest" }}
{{- if .Values.image.tag }}
{{- $tag = .Values.image.tag }}
{{- else if .Chart.AppVersion }}
{{- $tag = .Chart.AppVersion }}
{{- end }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

