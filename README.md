Golang API boilerplate
================

Go Server/API boilerplate using best practices, DDD, CQRS, ES.

Key concepts:
1. Rest API
2. [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design)  (DDD)
3. [CQRS](https://martinfowler.com/bliki/CQRS.html)
4. [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
5. [Docker](https://www.docker.com/what-docker)

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
### Prerequisites
In order to run this project you need to have Docker > 1.17.05 installed for building the production image.

### Repository structure
Repository holds two top-level directories, `pkg` and `cmd`.

`pkg` directory contains directories for each of libraries.

`cmd` directory contains directories for each of binaries.

### Configuration
Create your local `.env` file from `dist.env` files.

For each of binaries when building a docker image the enviroment variable file will be passed. This repository contains example `.env` file for `server` binary. There are always two files `.server.env` containing local configurations and is to be git ignored where [dist.server.env](dist.server.env) contains versioned example of configuration.

### Development
To setup development enviroment simply run [docker-compose](https://docs.docker.com/compose/gettingstarted/) command. The containers will be set up for each binaries and other services required for application to run.

### Deployment
**Release all binaries**
```sh
$> make all-release
```
**Release single binary**
```sh
$> make release-server
```
Other available commands: `build`, `run`, `stop`, `rm`, `release`, `publish`, `publish-latest`, `publish-version`, `tag`, `tag-latest`, `tag-version`. All comands follow the same convention for all binaries add `all-` prefix, for one binary add `-%` sufix where `%` is a directory name underneath `cmd`. For more informations about commands check [Makefile](Makefile).
