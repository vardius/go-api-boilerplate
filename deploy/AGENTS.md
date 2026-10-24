# deploy

How the services run outside `go run`: `compose/` holds files the root
`compose*.yaml` mount, `helm/go-api-boilerplate` is the Kubernetes chart.

- No secret has a value in this folder. The chart reads secrets from a
  Secret the operator creates (`existingSecret`); compose files carry only
  dev values named `dev-only-*`.
- Every container runs as a numeric non-root user with a read-only root
  filesystem and all capabilities dropped. A path that must be writable
  gets an `emptyDir`.
- An environment variable here must exist in the service's config (see
  `go run ./cmd/<service> env`). Add it to the service first.
- The store is a build: images are `-auth:<version>-<store>`. Migrations
  run once per release as a hook Job (`<service> migrate`), never on every
  pod start (`MIGRATE_ON_START` stays false in the chart).
- A change to the chart's values also changes `ci/*-values.yaml` and the
  deploy guide on the website.

Verify:

```bash
for s in memory mysql mongodb; do helm lint deploy/helm/go-api-boilerplate --strict -f deploy/helm/go-api-boilerplate/ci/$s-values.yaml; done
```

CI also renders each through kubeconform.
