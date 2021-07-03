🧰 Golang API Starter Kit
================
[![Build Status](https://travis-ci.com/vardius/go-api-boilerplate.svg?branch=master)](https://travis-ci.com/vardius/go-api-boilerplate)
![Test](https://github.com/vardius/go-api-boilerplate/workflows/Test/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/go-api-boilerplate)](https://goreportcard.com/report/github.com/vardius/go-api-boilerplate)
[![codecov](https://codecov.io/gh/vardius/go-api-boilerplate/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/go-api-boilerplate)
[![](https://godoc.org/github.com/vardius/go-api-boilerplate?status.svg)](https://pkg.go.dev/github.com/vardius/go-api-boilerplate)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/go-api-boilerplate/blob/master/LICENSE.md)
[![baker](https://opencollective.com/go-api-boilerplate/tiers/backer/badge.svg?label=backer&color=brightgreen)](https://opencollective.com/go-api-boilerplate/contribute/backer-10349/checkout)
[![sponsor](https://opencollective.com/go-api-boilerplate/tiers/sponsor/badge.svg?label=sponsor&color=brightgreen)](https://opencollective.com/go-api-boilerplate/contribute/sponsor-10350/checkout)

<img align="right" height="180px" src="https://github.com/vardius/gorouter/blob/master/website/src/static/img/logo.png?raw=true" alt="logo" />

Go Server/API boilerplate using best practices, DDD, CQRS, ES, gRPC.

<details>
  <summary>Table of Contents</summary>

<!-- toc -->

- [About](#about)
- [Documentation](#documentation)
- [Example](#example)
  - [Quick start](#quick-start)
    - [Build release](#build-release)
        - [Local image](#local-image)
        - [GitHub Package Registry](#github-package-registry)
        - [Private Registry](#private-registry)
    - [Deploy release](#build-release)
  - [Dashboard](#dashboard)
  - [Domain](#domain)
    - [Dispatching command](#dispatching-command)
  - [View](#view)
    - [Public routes](#public-routes)
    - [Protected routes](#protected-routes)
- [Sponsoring](#sponsoring)
<!-- tocstop -->

</details>

📖 ABOUT
==================================================
The main purpose of this project is to provide boilerplate project setup using best practices, DDD, CQRS, ES, gRPC. Featuring kubernetes for both development and production environments. Allowing to work with environment reflecting production one, allowing to reduce any misconfigurations.

This is mono-repository of many services such as authentication or user domain. Each service has it own code base with exception of shared packages to simplify things for this boilerplate. Services communicate witch each other using gRPC. Each service might expose HTTP API for external communication or/and gRPC.

This project setup should reduce the time spent on environment configuration for the whole kubernetes cluster and/or each of microservice. Extracting each of services to own repository or keeping it as mono-repo should be a matter of preference.

*Please look for comments like `@TODO` and `@FIXME` to better understand things than need attention.*

### Web UI example (React)
This boilerplate includes simple Web UI to demonstrate example interaction with API.
Once deployed and hosts are set please visit [https://api.go-api-boilerplate.local](https://api.go-api-boilerplate.local) to access UI.

<img align="center" src="https://github.com/vardius/go-api-boilerplate/blob/master/.github/Web%20UI.jpg?raw=true" alt="Web UI" />

### Key concepts:
1. Rest API
2. [Docker](https://www.docker.com/what-docker)
3. [Kubernetes](https://kubernetes.io/)
4. [Helm chart](https://helm.sh/)
5. [Terraform](https://terraform.io/)
6. [gRPC](https://grpc.io/docs/)
7. [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design) (DDD)
8. [CQRS](https://martinfowler.com/bliki/CQRS.html)
9. [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
10. [Hexagonal, Onion, Clean Architecture](https://herbertograca.com/2017/11/16/explicit-architecture-01-ddd-hexagonal-onion-clean-cqrs-how-i-put-it-all-together/)
11. [oAuth2](https://github.com/go-oauth2/oauth2)

Worth getting to know packages used in this boilerplate:
1. [gorouter](https://github.com/vardius/gorouter)
2. [message-bus](https://github.com/vardius/message-bus)
3. [gollback](https://github.com/vardius/gollback)
4. [shutdown](https://github.com/vardius/shutdown)
5. [pubsub](https://github.com/vardius/pubsub)
6. [pushpull](https://github.com/vardius/pushpull)
7. [gocontainer](https://github.com/vardius/gocontainer)

📚 DOCUMENTATION
==================================================

* [Wiki](https://github.com/vardius/go-api-boilerplate/wiki)
* [Package level docs](https://godoc.org/github.com/vardius/go-api-boilerplate#pkg-subdirectories)
* [Getting Started](https://github.com/vardius/go-api-boilerplate/wiki/1.-Getting-Started)
* [Installing and Setting up](https://github.com/vardius/go-api-boilerplate/wiki/2.-Installing-and-Setting-up)
* [Configuration](https://github.com/vardius/go-api-boilerplate/wiki/3.-Configuration)
* [Guides](https://github.com/vardius/go-api-boilerplate/wiki/4.-Guides)

🏫 EXAMPLE
==================================================
## Quick start

### Localhost alias
Edit `/etc/hosts` to add localhost alias
```bash
➜ go-api-boilerplate git:(master) cat /etc/hosts
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting. Do not change this entry.
##
127.0.0.1 go-api-boilerplate.local api.go-api-boilerplate.local maildev.go-api-boilerplate.local mysql.go-api-boilerplate.local
```
### Build release
#### Local image
```sh
make docker-build BIN=auth
make docker-build BIN=migrate
make docker-build BIN=user
make docker-build BIN=web
```
#### GitHub Package Registry
Creating tag with metadata will trigger [github workflow](https://github.com/vardius/go-api-boilerplate/actions) and publish docker image to GitHub Package Registry.

Tag `v1.0.0+user` will trigger build for `user` service releasing `1.0.0` docker image tag.
you can create release for all services in `cmd` directory.
```sh
v1.0.0+auth
v1.0.0+user
v1.0.0+web
v1.0.0+migrate
```

Replace image details in [main.yaml](cmd/user/main.yaml)
```diff
  image:
-    repository: go-api-boilerplate-user
+    repository: docker.pkg.github.com/vardius/go-api-boilerplate/go-api-boilerplate-user
-    tag: latest
+    tag: 1.0.0
    pullPolicy: IfNotPresent
```
repeat for all services and `migrate` init containers.
#### Private Registry
[Log in to Docker](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#log-in-to-docker)
```shell
docker login
```
Copy docker config
```shell
cp ~/.docker/config.json ./k8s/.docker/config.json
```
Verify [config.json](k8s/.docker/config.json)
### Deploy release
```shell
make terraform-install
```
#### Destroy
```shell
make terraform-destroy
```
If persistent volume is stack in terminating, this happens when persistent volume is protected. You should be able to cross verify this:
```shell
kubectl describe pvc PVC_NAME --namespace=go-api-boilerplate | grep Finalizers

Output:
Finalizers:    [kubernetes.io/pvc-protection]
```
You can fix this by setting finalizers to null using kubectl patch:
```shell
kubectl patch pvc PVC_NAME --namespace=go-api-boilerplate -p '{"metadata":{"finalizers": []}}' --type=merge
```
## Build tags
Build flags are used for different persistence layers. Please see `services.go` file for details. Provided layers are `mysql`, `mongo` and `memory`.
If desired in similar way new layer can be easily added, following given patter.

```shell
go build -tags=persistence_mysql
```

### Available build tags
- persistence_mysql (mysql service container)
- persistence_mongodb (mongodb service container)

**Important**
persistence layer defaults to memory if no flag is provided (Docker image sets persistence_mysql flag), see each service Dockerfile for details.

## Domain
### Dispatching command
Send example JSON via POST request
```sh
curl -d '{"email":"test@test.com"}' -H "Content-Type: application/json" -X POST https://api.go-api-boilerplate.local/users/v1/dispatch/user/user-register-with-email --insecure
```
## View
### Public routes
Get user details [https://api.go-api-boilerplate.local/users/v1/34e7ed39-aa94-4ef2-9422-401bba9fc812](https://api.go-api-boilerplate.local/users/v1/34e7ed39-aa94-4ef2-9422-401bba9fc812)
```json
{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}
```
Get list of users [https://api.go-api-boilerplate.local/users/v1?page=1&limit=10](https://api.go-api-boilerplate.local/users/v1?page=1&limit=10)
```json
{"page":1,"limit":20,"total":1,"users":[{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}]}
```
### Protected routes
Access protected route using auth token [https://api.go-api-boilerplate.local/users/v1/me](https://api.go-api-boilerplate.local/users/v1/me).
```json
{"code": "401","message": "Unauthorized"}
```
Request access token for user
```sh
curl -d '{"email":"test@test.com"}' -H "Content-Type: application/json" -X POST https://api.go-api-boilerplate.local/users/v1/dispatch/user/user-request-access-token --insecure
```
Get your access token from mail catcher [https://maildev.go-api-boilerplate.local](https://maildev.go-api-boilerplate.local).

Access protected route using auth token [https://api.go-api-boilerplate.local/users/v1/me?authToken=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyXHUwMDE277-977-977-977-9IiwiZXhwIjoxNTU5NjEwOTc2LCJzdWIiOiIzNGU3ZWQzOS1hYTk0LTRlZjItOTQyMi00MDFiYmE5ZmM4MTIifQ.pEkgtDAvNh2D3Dtgfpu4tt-Atn1h6QwMkDhz4KpgFxNX8jE7fQH00J6K5V7CV063pigxWhOMMTRLmQdhzhajzQ](https://api.go-api-boilerplate.local/users/v1/me?authToken=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyXHUwMDE277-977-977-977-9IiwiZXhwIjoxNTU5NjEwOTc2LCJzdWIiOiIzNGU3ZWQzOS1hYTk0LTRlZjItOTQyMi00MDFiYmE5ZmM4MTIifQ.pEkgtDAvNh2D3Dtgfpu4tt-Atn1h6QwMkDhz4KpgFxNX8jE7fQH00J6K5V7CV063pigxWhOMMTRLmQdhzhajzQ)
```json
{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}
```

💲 Sponsoring
==================================================

<p align="center">
  <a href="https://opencollective.com/go-api-boilerplate/contribute/sponsor-10349/checkout"><img src="https://opencollective.com/go-api-boilerplate/tiers/backer.svg"></a>
  <a href="https://opencollective.com/go-api-boilerplate/contribute/sponsor-10350/checkout"><img src="https://opencollective.com/go-api-boilerplate/tiers/sponsor.svg"></a>
</p>

## 🚀 Contributing

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/go-api-boilerplate/issues) to manage them.

## 👨🏻‍💻👩🏾‍💻 Core Team:

<table>
  <tbody>
    <tr>
      <td align="center" valign="top">
        <img width="150" height="150" src="https://github.com/vardius.png?s=150">
        <br>
        <a href="http://rafallorenz.com">Rafał Lorenz</a>
      </td>
      <td align="center" valign="top">
        <img width="150" height="150" src="https://github.com/mar1n3r0.png?s=150">
        <br>
        <a href="https://github.com/mar1n3r0">Marin Kirkov</a>
      </td>
   </tr>
  </tbody>
</table>

## 👥 Backers

Support us with a monthly donation and help us continue our activities.

<a href="https://opencollective.com/go-api-boilerplate/backer/0/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/0/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/1/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/1/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/2/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/2/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/3/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/3/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/4/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/4/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/5/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/5/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/6/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/6/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/7/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/7/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/8/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/8/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/9/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/9/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/10/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/10/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/11/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/11/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/12/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/12/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/13/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/13/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/14/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/14/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/15/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/15/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/16/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/16/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/17/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/17/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/18/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/18/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/19/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/19/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/20/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/20/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/21/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/21/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/22/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/22/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/23/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/23/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/24/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/24/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/25/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/25/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/26/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/26/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/27/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/27/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/28/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/28/avatar.svg"></a>
<a href="https://opencollective.com/go-api-boilerplate/backer/29/website" target="_blank"><img src="https://opencollective.com/go-api-boilerplate/backer/29/avatar.svg"></a>

## 🥇 Sponsors

Proudly sponsored by [Open Collective sponsors](https://opencollective.com/go-api-boilerplate#sponsor).

- 👥 [Contribute on Open Collective](https://opencollective.com/go-api-boilerplate#sponsor)

## 📜 [License](LICENSE.md)
