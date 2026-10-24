# go-api-boilerplate chart

Runs the auth and user services and the web app. The web pods serve the
single-page app and proxy `/api/auth` and `/api/user` to the services, so
the route below points at one Service and the browser sees one origin.

The database is yours: a managed MySQL 8.4 or MongoDB replica set. The
`memory` store is for trying the chart and allows one replica per service.

## Install

Create the auth service's signing key and the Secrets the chart reads:

```bash
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out signing-key.pem
kubectl create secret generic gab-auth --from-file=signing-key.pem --from-literal=mysql-password=…
kubectl create secret generic gab-user --from-literal=smtp-password=… --from-literal=mysql-password=…
```

Then install with your values; `ci/mysql-values.yaml` is a complete example:

```bash
helm install gab deploy/helm/go-api-boilerplate -f my-values.yaml
```

With `store: mysql` or `mongodb`, a hook Job runs each service's
`migrate` command before the Deployments roll.

## Rotating the signing key

Put the old key's public half in a Secret, set `auth.previousKeysSecret`
to it, and replace `signing-key.pem`. Tokens signed with the old key keep
verifying until they expire (15 minutes); then remove the old key.

```bash
openssl pkey -in old-signing-key.pem -pubout -out old.pem
kubectl create secret generic gab-auth-previous --from-file=old.pem
```

Every value is described in [values.yaml](values.yaml).
