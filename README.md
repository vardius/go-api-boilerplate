Golang API Starter Kit
================
[![Build Status](https://travis-ci.org/vardius/go-api-boilerplate.svg?branch=master)](https://travis-ci.org/vardius/go-api-boilerplate)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/go-api-boilerplate)](https://goreportcard.com/report/github.com/vardius/go-api-boilerplate)
[![codecov](https://codecov.io/gh/vardius/go-api-boilerplate/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/go-api-boilerplate)
[![](https://godoc.org/github.com/vardius/go-api-boilerplate?status.svg)](http://godoc.org/github.com/vardius/go-api-boilerplate)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/go-api-boilerplate/blob/master/LICENSE.md)

Go Server/API boilerplate using best practices, DDD, CQRS, ES, gRPC.

<details>
  <summary>Table of Contents</summary>

<!-- toc -->

- [About](#about)
- [How to use](#how-to-use)
  - [Prerequisites](#prerequisites)
    - [Localhost alias](#localhost-alias)
    - [Helm charts](#helm-charts)
  - [Makefile](#makefile)
  - [Running](#running)
    - [Build docker image](#step-1-build-docker-image)
    - [Deploy](#step-2-deploy)
    - [Debug](#step-3-debug)
  - [Kubernetes](#kubernetes)
  - [Vendor](#vendor)
- [Documentation](#documentation)
- [Example](#example)
  - [Domain](#domain)
    - [Dispatching command](#dispatching-command)
  - [View](#view)
    - [Public routes](#public-routes)
    - [Protected routes](#protected-routes)
<!-- tocstop -->

</details>

ABOUT
==================================================
This repository was created for personal use and needs, may contain bugs. If found please report. If you think some things could be done better, or if this repository is missing something feel free to contribute and create pull request.

Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/go-api-boilerplate/issues) to manage them.

![Dashboard](../master/.github/kubernetes-dashboard-overview.png)
![Dashboard](../master/.github/kubernetes-dashboard-pods.png)

Key concepts:
1. Rest API
2. [Docker](https://www.docker.com/what-docker)
3. [Kubernetes](https://kubernetes.io/)
4. [Helm chart](https://helm.sh/)
5. [gRPC](https://grpc.io/docs/)
6. [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design)  (DDD)
7. [CQRS](https://martinfowler.com/bliki/CQRS.html)
8. [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
9. [Hexagonal, Onion, Clean Architecture](https://herbertograca.com/2017/11/16/explicit-architecture-01-ddd-hexagonal-onion-clean-cqrs-how-i-put-it-all-together/)
10. [oAuth2](https://github.com/go-oauth2/oauth2)

Worth getting to know packages used in this boilerplate:
1. [gorouter](https://github.com/vardius/gorouter)
2. [message-bus](https://github.com/vardius/message-bus)
3. [gollback](https://github.com/vardius/gollback)
4. [shutdown](https://github.com/vardius/shutdown)
5. [pubsub](https://github.com/vardius/pubsub)

HOW TO USE
==================================================

## Getting started
### Prerequisites
In order to run this project you need to have Docker > 1.17.05 for building the production image and Kubernetes cluster > 1.11 for running pods installed.
#### Localhost alias
For examples below to work, edit `/etc/hosts` to add localhost alias
```bash
➜  go-api-boilerplate git:(master) cat /etc/hosts
##
# Host Database
#
# localhost is used to configure the loopback interface
# when the system is booting.  Do not change this entry.
##
127.0.0.1       localhost go-api-boilerplate.local
```
#### Helm charts
Helm chart are used to automate the application deployment in a Kubernetes cluster. Once the application is deployed and working, it also explores how to modify the source code for publishing a new application release and how to perform rolling updates in Kubernetes using the Helm CLI.

To deploy application on Kubernetes using Helm you will typically follow these steps:

1. Add application to cmd directory
2. Build the Docker image
3. Publish the Docker image
4. Create the Helm Chart (extend microservice chart)
5. Deploy the application in Kubernetes
6. Update the source code and the Helm chart

[Install And Configure Helm And Tiller](https://docs.helm.sh/using_helm/#install-helm)
### Makefile
```bash
➜  go-api-boilerplate git:(master) ✗ make help
version                        Show version
key                            [HTTP] Generate key
cert                           [HTTP] Generate self signed certificate
docker-build                   [DOCKER] Build given container. Example: `make docker-build BIN=user`
docker-run                     [DOCKER] Run container on given port. Example: `make docker-run BIN=user PORT=3000`
docker-stop                    [DOCKER] Stop docker container. Example: `make docker-stop BIN=user`
docker-rm                      [DOCKER] Stop and then remove docker container. Example: `make docker-rm BIN=user`
docker-publish                 [DOCKER] Docker publish. Example: `make docker-publish BIN=user REGISTRY=https://your-registry.com`
docker-tag                     [DOCKER] Tag current container. Example: `make docker-tag BIN=user REGISTRY=https://your-registry.com`
docker-release                 [DOCKER] Docker release - build, tag and push the container. Example: `make docker-release BIN=user REGISTRY=https://your-registry.com`
helm-install                   [HELM] Deploy the Helm chart for application. Example: `make helm-install`
helm-upgrade                   [HELM] Update the Helm chart for application. Example: `make helm-upgrade`
helm-history                   [HELM] See what revisions have been made to the application's helm chart. Example: `make helm-history`
helm-dependencies              [HELM] Update helm chart's dependencies for application. Example: `make helm-dependencies`
helm-delete                    [HELM] Delete helm chart for application. Example: `make helm-delete`
telepresence-swap-local        [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local process. Example: `make telepresence-swap-local BIN=user PORT=3000 DEPLOYMENT=go-api-boilerplate-user`
telepresence-swap-docker       [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local docker image. Example: `make telepresence-swap-docker BIN=user PORT=3000 DEPLOYMENT=go-api-boilerplate-user`
aws-repo-login                 [HELPER] login to AWS-ECR
```
### Running
To run services repeat following steps for each service defined in [`./cmd/`](../master/cmd) directory. Changing `BIN=` value to directory name from `./cmd/{service_name}` path.
#### STEP 1. Build docker image
```bash
make docker-build BIN=user
```
#### STEP 2. Deploy
In order to [install a cert-manager](https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html#installing-with-helm) Helm chart, you must run
 - Install cert-manager resources
 ```bash
 # Install the CustomResourceDefinition resources separately
kubectl apply -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.8/deploy/manifests/00-crds.yaml
```
 - Add external charts repositories
```bash
# Add the Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io
# Update your local Helm chart repository cache
helm repo update
```
 - Install dependencies
```bash
make helm-dependencies
```
 - Deploy to kubernetes cluster
```bash
make helm-install
```
This will deploy every service to the kubernetes cluster using your local docker image (built in the first step).

Each service defined in [`./cmd/`](../master/cmd) directory should contain helm chart (extension of microservice chart) later on referenced in app chart requirements alongside other external services required to run complete environment like `mysql`. For more details please see [helm chart](../master/helm/app/requirements.yaml)
#### STEP 3. Debug
To debug deployment you can simply use [telepresence](https://www.telepresence.io/reference/install) and swap kubernetes deployment for local go service or local docker image. For example to swap for local docker image run:
```sh
make telepresence-swap-docker BIN=user PORT=3001 DEPLOYMENT=go-api-boilerplate-user
```
This command should swap deployment giving similar output to the one below:
```
➜  go-api-boilerplate git:(master) ✗ make telepresence-swap-docker BIN=user PORT=3001 DEPLOYMENT=go-api-boilerplate-user
telepresence \
	--swap-deployment go-api-boilerplate-user \
	--docker-run -i -t --rm -p=3001:3001 --name="user" user:latest
T: Volumes are rooted at $TELEPRESENCE_ROOT. See https://telepresence.io/howto/volumes.html for
T: details.
2019/01/10 06:24:37.963250 INFO:  [CommandBus|Subscribe]: *user.RegisterWithEmail
2019/01/10 06:24:37.963332 INFO:  [CommandBus|Subscribe]: *user.RegisterWithGoogle
2019/01/10 06:24:37.963357 INFO:  [CommandBus|Subscribe]: *user.RegisterWithFacebook
2019/01/10 06:24:37.963428 INFO:  [CommandBus|Subscribe]: *user.ChangeEmailAddress
2019/01/10 06:24:37.963445 INFO:  [EventBus|Subscribe]: *user.WasRegisteredWithEmail
2019/01/10 06:24:37.963493 INFO:  [EventBus|Subscribe]: *user.WasRegisteredWithGoogle
2019/01/10 06:24:37.963540 INFO:  [EventBus|Subscribe]: *user.WasRegisteredWithFacebook
2019/01/10 06:24:37.963561 INFO:  [EventBus|Subscribe]: *user.EmailAddressWasChanged
2019/01/10 06:24:37.964452 INFO: running at 0.0.0.0:3000
^C
2019/01/10 06:30:16.266108 INFO: shutting down...
2019/01/10 06:30:16.283392 INFO: gracefully stopped
T: Exit cleanup in progress
# --docker-run --rm -it -v -p=3001:3001 user:latest
```
### Kubernetes
The [Dashboard UI](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/) is accessible at [https://go-api-boilerplate.local/dashboard](https://go-api-boilerplate.local/dashboard/#!/overview?namespace=go-api-boilerplate) thanks to kubernetes-dashboard [helm chart](https://github.com/helm/charts/tree/master/stable/kubernetes-dashboard). To see available tokens for login run:
```bash
kubectl -n kube-system describe secret $(kubectl -n kube-system get secret | grep admin-user | awk '{print $1}')
```
To remove release run:
```bash
make helm-delete
```
### Vendor
Build the module. This will automatically add missing or unconverted dependencies as needed to satisfy imports for this particular build invocation
```bash
go build ./...
```
For more read: https://github.com/golang/go/wiki/Modules

DOCUMENTATION
==================================================

* [Wiki](https://github.com/vardius/go-api-boilerplate/wiki)
* [Package level docs](https://godoc.org/github.com/vardius/go-api-boilerplate#pkg-subdirectories)

EXAMPLE
==================================================

## Domain
### Dispatching command
Send example JSON via POST request
```sh
curl -sSL -D -d '{"email":"test@test.com"}' -H "Content-Type: application/json" -X POST https://go-api-boilerplate.local/users/dispatch/register-user-with-email -o /dev/null --insecure
```
Request access token for user
```sh
curl -sSL -D -d '{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812"}' -H "Content-Type: application/json" -X POST https://go-api-boilerplate.local/users/dispatch/request-user-access-token -o /dev/null --insecure
```
## View
### Public routes
Get user details [https://go-api-boilerplate.local/users/34e7ed39-aa94-4ef2-9422-401bba9fc812](https://go-api-boilerplate.local/users/34e7ed39-aa94-4ef2-9422-401bba9fc812)
```json
{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}
```
Get list of users [https://go-api-boilerplate.local/users?page=1&limit=10](https://go-api-boilerplate.local/users?page=1&limit=10)
```json
{"page":1,"limit":20,"total":1,"users":[{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}]}
```
### Protected routes
Access protected route using auth token [https://go-api-boilerplate.local/users/me](https://go-api-boilerplate.local/users/me). Get your access token from user pod event handler logs.
```json
{"code": "401","message": "Unauthorized"}
```
Access protected route using auth token [https://go-api-boilerplate.local/users/me?authToken=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyXHUwMDE277-977-977-977-9IiwiZXhwIjoxNTU5NjEwOTc2LCJzdWIiOiIzNGU3ZWQzOS1hYTk0LTRlZjItOTQyMi00MDFiYmE5ZmM4MTIifQ.pEkgtDAvNh2D3Dtgfpu4tt-Atn1h6QwMkDhz4KpgFxNX8jE7fQH00J6K5V7CV063pigxWhOMMTRLmQdhzhajzQ](https://go-api-boilerplate.local/users/me?authToken=eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJyXHUwMDE277-977-977-977-9IiwiZXhwIjoxNTU5NjEwOTc2LCJzdWIiOiIzNGU3ZWQzOS1hYTk0LTRlZjItOTQyMi00MDFiYmE5ZmM4MTIifQ.pEkgtDAvNh2D3Dtgfpu4tt-Atn1h6QwMkDhz4KpgFxNX8jE7fQH00J6K5V7CV063pigxWhOMMTRLmQdhzhajzQ)
```json
{"id":"34e7ed39-aa94-4ef2-9422-401bba9fc812","email":"test@test.com"}
```
