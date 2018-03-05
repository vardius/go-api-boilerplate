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

![Screenshot](../master/_layouts/startup.png)

Key concepts:
1. Rest API
2. [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design)  (DDD)
3. [CQRS](https://martinfowler.com/bliki/CQRS.html)
4. [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
5. [Docker](https://www.docker.com/what-docker)
5. [gRPC](https://grpc.io/docs/)

Worth getting to know packages used in this boilerplate:
1. [gorouter](https://github.com/vardius/gorouter)
2. [message-bus](https://github.com/vardius/message-bus)
3. [env](https://github.com/caarlos0/env)

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
### [Documentation](https://github.com/vardius/go-api-boilerplate/wiki)
### Packages
___
* [domain](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/domain)
* [recover](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/recover)
* [log](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/log)
* [jwt](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/jwt)
* [socialmedia](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/socialmedia)
___
* [http/response](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/http/response)
___
* [security/authenticator](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/security/authenticator)
* [security/firewall](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/security/firewall)
* [security/identity](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/security/identity)
___
* [memory/commandbus](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/memory/commandbus)
* [memory/eventbus](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/memory/eventbus)
* [memory/eventstore](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/memory/eventstore)
___
* [os/shutdown](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/os/shutdown)
___
* [aws/dynamodb/commandbus](https://godoc.org/github.com/vardius/go-api-boilerplate/pkg/aws/dynamodb/commandbus)
### Prerequisites
In order to run this project you need to have Docker > 1.17.05 installed for building the production image.
### Vendor
To update vendors run
```bash
go get -u github.com/golang/dep/cmd/dep
dep init
dep ensure -update
```
