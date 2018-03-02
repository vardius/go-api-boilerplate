/*
Package goapiboilerplate provides Go Server/API boilerplate using best practices, DDD, CQRS, ES.

Directory Layout:
  .
  ├── /.vscode/            # Visual Studio Code remote debugging setttings
  ├── /nginx/              # Nginx docker container configuration
  ├── /cmd/                # Binaries
  │   ├── /userserver/     # User service server binary
  │   │   └── /main.go     # User domain grpc server
  │   │   └── /.env        # Binary environment configuration
  │   ├── /apiserver/      # API Server binary
  │   │   └── /main.go     # API Server application - glues together libraries
  │   │   └── /.env        # Binary environment configuration
  │   ├── /...             # etc.
  ├── /internal/           # Internal libraries
  │   ├── /user/           # User bounded context
  │   │   ├── /domain/     # User domain
  │   │   ├── /server/     # User server implementation
  │   │   ├── /client/     # User client implementation
  ├── /pkg/                # Libraries
  │   ├── /proto/          # Package proto is a generated protocol buffer package.
  │   ├── /auth/           # Authorization tools
  │   ├── /domain/         # Domain libraries
  │   ├── /http/           # Http utils
  │   ├── /...             # etc.
  ├── /vendor/             # Vendor libraries
  ├── docker-compose.yml   # Defines Docker services, networks and volumes per developer environment
  ├── Dockerfile           # Docker image for production
  ├── Makefile             # Commands for building a Docker image for production and deployment
  ├── .env                 # Project environment configuration
*/
package goapiboilerplate
