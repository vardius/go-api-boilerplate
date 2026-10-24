# go-api-boilerplate chart

Runs the auth and user services and the web app. The web pods serve the
single-page app and proxy `/api/auth` and `/api/user` to the services, so
the route below points at one Service and the browser sees one origin.

The database is yours: a managed MySQL 9.7 or MongoDB replica set. The
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

Services cache the auth service's public keys for up to an hour, so publish
a new key before signing with it. The auth service reads its keys only when
it starts, and changing a Secret doesn't restart it, so each step ends with
a restart:

1. Add the new key's public half to the `auth.previousKeysSecret` Secret
   and restart auth. The JWKS now lists it.
2. Wait more than an hour, so every service has fetched it.
3. Put the new private key in `signing-key.pem`, add the old key's public
   half to `auth.previousKeysSecret`, and restart auth. A previous key that
   is also the signing key is skipped, so `new.pem` can stay.
4. After another hour, remove the old public key and restart auth.

```bash
openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out new-signing-key.pem
openssl pkey -in new-signing-key.pem -pubout -out new.pem
kubectl create secret generic gab-auth-previous --from-file=new.pem --dry-run=client -o yaml | kubectl apply -f -
# The first time, also set auth.previousKeysSecret=gab-auth-previous and upgrade.
kubectl rollout restart deployment/gab-auth
```

Every value is described in [values.yaml](values.yaml).
