Golang API Starter Kit
================
[![Build Status](https://travis-ci.org/vardius/go-api-boilerplate.svg?branch=master)](https://travis-ci.org/vardius/go-api-boilerplate)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/go-api-boilerplate)](https://goreportcard.com/report/github.com/vardius/go-api-boilerplate)
[![codecov](https://codecov.io/gh/vardius/go-api-boilerplate/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/go-api-boilerplate)
[![](https://godoc.org/github.com/vardius/go-api-boilerplate?status.svg)](http://godoc.org/github.com/vardius/go-api-boilerplate)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/go-api-boilerplate/blob/master/LICENSE.md)
[![Beerpay](https://beerpay.io/vardius/go-api-boilerplate/badge.svg?style=beer-square)](https://beerpay.io/vardius/go-api-boilerplate)
[![Beerpay](https://beerpay.io/vardius/go-api-boilerplate/make-wish.svg?style=flat-square)](https://beerpay.io/vardius/go-api-boilerplate?focus=wish)

Go Server/API boilerplate using best practices, DDD, CQRS, ES, gRPC.

![Dashboard](../master/.github/kubernetes-dashboard.png)

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

Worth getting to know packages used in this boilerplate:
1. [gorouter](https://github.com/vardius/gorouter)
2. [message-bus](https://github.com/vardius/message-bus)

ABOUT
==================================================
This repository was created for personal use and needs, may contain bugs. If found please report. If you think some things could be done better, or if this repository is missing something feel free to contribute and create pull request.

Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/go-api-boilerplate/issues) to manage them.

HOW TO USE
==================================================

## Getting started
### Prerequisites
In order to run this project you need to have Docker > 1.17.05 for building the production image and Kubernetes cluster > 1.11 for running pods installed.
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
helm-dependencies-cmd          [HELM] Update helm chart's dependencies for microservice. Example: `make helm-dependencies-cmd BIN=user`
telepresence-swap-local        [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local process. Example: `make telepresence-swap-local BIN=user PORT=3000 DEPLOYMENT=go-api-boilerplate-user`
telepresence-swap-docker       [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local docker image. Example: `make telepresence-swap-docker BIN=user PORT=3000 DEPLOYMENT=go-api-boilerplate-user`
aws-repo-login                 [HELPER] login to AWS-ECR
```
### Kubernetes
The [Dashboard UI](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/) is not deployed by default. To deploy it, run the following command:
```bash
kubectl create -f https://raw.githubusercontent.com/kubernetes/dashboard/master/aio/deploy/recommended/kubernetes-dashboard.yaml
```
You can access Dashboard using the kubectl command-line tool by running the following command:
```bash
kubectl proxy
```
Kubectl will handle authentication and make Dashboard available at http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/.
The UI can only be accessed from the machine where the command is executed. See kubectl proxy --help for more options.
### Helm charts
Helm chart are used to automate the application deployment in a Kubernetes cluster. Once the application is deployed and working, it also explores how to modify the source code for publishing a new application release and how to perform rolling updates in Kubernetes using the Helm CLI.

To deploy application on Kubernetes using Helm you will typically follow these steps:

Step 1: Add application to cmd directory
Step 2: Build the Docker image
Step 3: Publish the Docker image
Step 4: Create the Helm Chart (extend microservice chart)
Step 5: Deploy the application in Kubernetes
Step 6: Update the source code and the Helm chart

[Install And Configure Helm And Tiller](https://docs.bitnami.com/kubernetes/get-started-kubernetes/#step-4-install-helm-and-tiller)
### Vendor
Build the module. This will automatically add missing or unconverted dependencies as needed to satisfy imports for this particular build invocation
```bash
go build ./...
```
For more read: https://github.com/golang/go/wiki/Modules
### Running
To run services repeat following steps for each service defined in [`./cmd/`](../master/cmd) directory. Changing `BIN=` value to directory name from `./cmd/{service_name}` path.
#### STEP 1. Build docker image
```bash
make docker-build BIN=user
```
#### STEP 2. Deploy
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
2019/01/10 06:24:37.963250 INFO:  [user CommandBus|Subscribe]: *user.RegisterWithEmail
2019/01/10 06:24:37.963332 INFO:  [user CommandBus|Subscribe]: *user.RegisterWithGoogle
2019/01/10 06:24:37.963357 INFO:  [user CommandBus|Subscribe]: *user.RegisterWithFacebook
2019/01/10 06:24:37.963428 INFO:  [user CommandBus|Subscribe]: *user.ChangeEmailAddress
2019/01/10 06:24:37.963445 INFO:  [user EventBus|Subscribe]: *user.WasRegisteredWithEmail
2019/01/10 06:24:37.963493 INFO:  [user EventBus|Subscribe]: *user.WasRegisteredWithGoogle
2019/01/10 06:24:37.963540 INFO:  [user EventBus|Subscribe]: *user.WasRegisteredWithFacebook
2019/01/10 06:24:37.963561 INFO:  [user EventBus|Subscribe]: *user.EmailAddressWasChanged
2019/01/10 06:24:37.964452 INFO:  [user] running at 0.0.0.0:3000
^C
2019/01/10 06:30:16.266108 INFO:  [user] shutting down...
2019/01/10 06:30:16.283392 INFO:  [user] gracefully stopped
T: Exit cleanup in progress
# --docker-run --rm -it -v -p=3001:3001 user:latest
```
#### STEP 4. Test example
Send example JSON via POST request
```sh
curl -d '{"email":"test@test.com"}' -H "Content-Type: application/json" -X POST http://localhost:3000/users/dispatch/register-user-with-email
```
**proxy** pod logs should look something like:
```sh
2019/01/06 09:37:52.453329 INFO:  [POST Request|Start]: /dispatch/register-user-with-email
2019/01/06 09:37:52.461655 INFO:  [POST Request|End] /dispatch/register-user-with-email 8.2233ms
```
**user** pod logs should look something like:
```sh
2019/01/06 09:37:52.459095 DEBUG: [user CommandBus|Publish]: *user.RegisterWithEmail &{Email:test@test.com}
2019/01/06 09:37:52.459966 DEBUG: [user EventBus|Publish]: *user.WasRegisteredWithEmail {"id":"4270a1ca-bfba-486a-946d-9d7b8a893ea2","email":"test@test.com"}
2019/01/06 09:37:52 [user EventHandler] {"id":"4270a1ca-bfba-486a-946d-9d7b8a893ea2","email":"test@test.com"}
```
### Documentation
* [Wiki](https://github.com/vardius/go-api-boilerplate/wiki)
* [Package level docs](https://godoc.org/github.com/vardius/go-api-boilerplate#pkg-subdirectories)
