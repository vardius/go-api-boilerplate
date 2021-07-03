---

apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-prod
  namespace: ${namespace}
  labels:
    ${indent(4, labels)}
  annotations:
    ${indent(4, annotations)}
spec:
  ca:
    secretName: wildcard-${replace(app, ".", "-")}-tls

%{ for domain in domains }
---

apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-${replace(domain.name, ".", "-")}-tls
  namespace: ${namespace}
  labels:
    ${indent(4, labels)}
  annotations:
    ${indent(4, annotations)}
spec:
  commonName: "*.${domain.name}"
  dnsNames:
    - "${domain.name}"
    - "*.${domain.name}"
  isCA: true
  secretName: wildcard-${replace(app, ".", "-")}-tls
  issuerRef:
    name: selfsigned-prod
    kind: ClusterIssuer
  # DCOS-60297 Update certificate to comply with Apple security requirements
  # https://support.apple.com/en-us/HT210176
  usages:
    - digital signature
    - key encipherment
    - server auth
    - code signing
%{ endfor }

---

apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: ${replace(app, ".", "-")}-ingress
  namespace: ${namespace}
  labels:
    ${indent(4, labels)}
  annotations:
    ${indent(4, annotations)}
    kubernetes.io/ingress.class: nginx
    kubernetes.io/tls-acme: "true"
    kubernetes.io/secure-backends: "true"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rewrite-target: /$2
    cert-manager.io/cluster-issuer: selfsigned-prod
spec:
  tls:
    - secretName: wildcard-${replace(app, ".", "-")}-tls
      hosts:
      %{ for domain in domains }
        - "${domain.name}"
        %{ for subdomain in domain.subdomains }
        - "${subdomain.name}.${domain.name}"
        %{ endfor }
      %{ endfor }
  rules:
  %{ for domain in domains }
    - host: "${domain.name}"
      http:
        paths:
        %{ for path in domain.paths }
        - path: "${path.path}"
          backend:
            serviceName: "${path.serviceName}"
            servicePort: ${path.servicePort}
        %{ endfor }
    %{ for subdomain in domain.subdomains }
    - host: "${subdomain.name}.${domain.name}"
      http:
        paths:
        %{ for path in subdomain.paths }
        - path: "${path.path}"
          backend:
            serviceName: "${path.serviceName}"
            servicePort: ${path.servicePort}
        %{ endfor }
    %{ endfor }
  %{ endfor }
