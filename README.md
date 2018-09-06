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
4. [gRPC](https://grpc.io/docs/)
5. [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design)  (DDD)
6. [CQRS](https://martinfowler.com/bliki/CQRS.html)
7. [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
8. [Hexagonal, Onion, Clean Architecture](https://herbertograca.com/2017/11/16/explicit-architecture-01-ddd-hexagonal-onion-clean-cqrs-how-i-put-it-all-together/)

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
kubernetes-create              [KUBERNETES] Create kubernetes deployment. Example: `make kubernetes-create BIN=user`
aws-repo-login                 [HELPER] login to AWS-ECR
```
### Kubernetes
The Dashboard UI is not deployed by default. To deploy it, run the following command:
```bash
kubectl create -f https://raw.githubusercontent.com/kubernetes/dashboard/master/src/deploy/recommended/kubernetes-dashboard.yaml
```
You can access Dashboard using the kubectl command-line tool by running the following command:
```bash
kubectl proxy
```
Kubectl will handle authentication with apiserver and make Dashboard available at http://localhost:8001/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/.
The UI can only be accessed from the machine where the command is executed. See kubectl proxy --help for more options.
### Vendor
Build the module. This will automatically add missing or unconverted dependencies as needed to satisfy imports for this particular build invocation
```bash
go build ./...
```
For more read: https://github.com/golang/go/wiki/Modules
### Running
To run services repeat following steps for each micro-service. Changing `BIN=` value to directory name from `./cmd` path.
### STEP 1. Build docker image
```bash
make docker-build BIN=user
```
### STEP 2. Deploy
```bash
make kubernetes-create BIN=user
```
This will deploy each of them to the kubernetes cluster using your local docker image (built in first step).
### Documentation
* [Wiki](https://github.com/vardius/go-api-boilerplate/wiki)
* [Package level docs](https://godoc.org/github.com/vardius/go-api-boilerplate#pkg-subdirectories)
