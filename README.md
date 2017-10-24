Golang API Starter Kit
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

#### Directory Layout
```bash
.
├── /.vscode/            # Visual Studio Code remote debugging setttings
├── /nginx/              # Nginx docker container configuration
├── /cmd/                # Binaries
│   ├── /server/         # Server binary
│   │   └── /main.go     # Server application - glues together libraries
│   ├── /...             # etc.
├── /pkg/                # Libraries
│   ├── /controller/     # Http handlers
│   │   ├── /dispatch.go # Dispatch command endpoint
│   │   ├── /...         # etc.
│   ├── /domain/         # Domain libraries
│   │   ├── /user/       # User domain
│   │   │   ├── /main.go # Main user domain entrypoint
│   │   ├── /...         # etc.
│   ├── /middleware/     # Http middlewares
│   │   ├── /cors.go     # CORS middleware
│   │   ├── /...         # etc.
├── /vendor/             # Vendor libraries
├── docker-compose.yml   # Defines Docker services, networks and volumes per developer environment
├── Dockerfile           # Docker image for production
├── Makefile             # Commands for building a Docker image for production and deployment
├── .env                 # Enviroment configuration
└── bootstart.sh         # Configuration script for docker containers
```

### Configuration
Create your local `.env` file from `dist.env` files.

For each of binaries when building a docker image the enviroment variable file will be passed. This repository contains example `.env` file for `server` binary. There are always two files `.server.env` containing local configurations and is to be git ignored where [dist.server.env](dist.server.env) contains versioned example of configuration.

## Development
To setup development enviroment simply run [docker-compose](https://docs.docker.com/compose/gettingstarted/) command. The containers will be set up for each binaries and other services required for application to run.

You can debug your program with [Delve](https://github.com/derekparker/delve) which is a debugger for the Go programming language running on port **2345**. Repository includes [VS Code](https://code.visualstudio.com/) settings to enable [remote dubbuging](https://github.com/Microsoft/vscode-go/wiki/Debugging-Go-code-using-VS-Code) within docker containers.

## Deployment
### Docker
Each binary will have its own docker container.
#### Build
Build the container(s)

**Build all binaries**
```sh
$> make all-build
```
**Build single binary**
```sh
$> make build-server
```
#### Run
Run container(s) on port configured in `.env`

**Run all binaries**
```sh
$> make all-run
```
**Run single binary**
```sh
$> make build-run
```
#### Release
build, tag and push the container(s)

**Release all binaries**
```sh
$> make all-release
```
**Release single binary**
```sh
$> make release-server
```
### Makefile
Available commands:

`build`, `run`, `stop`, `rm`, `release`, `publish`, `publish-latest`, `publish-version`, `tag`, `tag-latest`, `tag-version`

All comands follow the same convention for all binaries add `all-` prefix, for one binary add `-%` sufix where `%` is a directory name underneath `cmd`. For more informations about commands check [Makefile](Makefile).
