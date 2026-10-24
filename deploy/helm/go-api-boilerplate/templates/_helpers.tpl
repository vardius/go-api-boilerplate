{{- define "gab.fullname" -}}
{{- default .Release.Name .Values.fullnameOverride | trunc 50 | trimSuffix "-" -}}
{{- end -}}

{{- define "gab.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .root.Chart.Name .root.Chart.Version }}
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
app.kubernetes.io/version: {{ .root.Chart.AppVersion | quote }}
{{ include "gab.selector" . }}
{{- end -}}

{{- define "gab.selector" -}}
app.kubernetes.io/name: {{ .root.Chart.Name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{/* Go images are built per store: <repository>-<service>:<tag>-<store>. */}}
{{- define "gab.image" -}}
{{- $v := .root.Values -}}
{{- $tag := default .root.Chart.AppVersion $v.image.tag -}}
{{- if eq .component "web" -}}
{{ $v.image.registry }}/{{ $v.image.repository }}-web:{{ $tag }}
{{- else -}}
{{ $v.image.registry }}/{{ $v.image.repository }}-{{ .component }}:{{ $tag }}-{{ $v.store }}
{{- end -}}
{{- end -}}

{{- define "gab.validate" -}}
{{- if not (has .Values.store (list "memory" "mysql" "mongodb")) -}}
{{- fail "store must be memory, mysql or mongodb" -}}
{{- end -}}
{{- if and (eq .Values.store "memory") (or (gt (int .Values.auth.replicas) 1) (gt (int .Values.user.replicas) 1)) -}}
{{- fail "the memory store keeps data in the pod: run one replica per service, or pick mysql or mongodb" -}}
{{- end -}}
{{- if not .Values.auth.existingSecret -}}
{{- fail "auth.existingSecret is required: a Secret with signing-key.pem (see values.yaml)" -}}
{{- end -}}
{{- if not .Values.user.smtp.addr -}}
{{- fail "user.smtp.addr is required: the SMTP server for sign-in links" -}}
{{- end -}}
{{- end -}}

{{- define "gab.podSecurity" -}}
automountServiceAccountToken: false
securityContext:
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}

{{- define "gab.containerSecurity" -}}
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: [ALL]
{{- end -}}

{{/* The environment both the Deployment and the migration Job of a service need. */}}
{{- define "gab.env" -}}
{{- $root := .root -}}
{{- $v := $root.Values -}}
{{- $svc := index $v .component -}}
{{- if eq .component "auth" -}}
- name: APP_ENV
  value: production
{{ end -}}
- name: LOG_LEVEL
  value: {{ $v.logLevel | quote }}
- name: MIGRATE_ON_START
  value: "false"
{{- if eq $v.store "mysql" }}
- name: MYSQL_ADDR
  value: {{ required (printf "%s.mysql.addr is required with store: mysql" .component) $svc.mysql.addr | quote }}
- name: MYSQL_USER
  value: {{ $svc.mysql.user | quote }}
- name: MYSQL_DATABASE
  value: {{ $svc.mysql.database | quote }}
- name: MYSQL_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ required (printf "%s.existingSecret is required with store: mysql" .component) $svc.existingSecret }}
      key: mysql-password
{{- else if eq $v.store "mongodb" }}
- name: MONGODB_DATABASE
  value: {{ $svc.mongodb.database | quote }}
- name: MONGODB_URI
  valueFrom:
    secretKeyRef:
      name: {{ required (printf "%s.existingSecret is required with store: mongodb" .component) $svc.existingSecret }}
      key: mongodb-uri
{{- end }}
{{- end -}}
