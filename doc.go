/*
Package goapiboilerplate provides Go Server/API boilerplate using best practices, DDD, CQRS, ES.

Directory Layout:
  .
  ├── /.vscode/            # Visual Studio Code remote debugging setttings
  ├── /nginx/              # Nginx docker container configuration
  ├── /cmd/                # Binaries
  │   ├── /server/         # Server binary
  │   │   └── /main.go     # Server application - glues together libraries
  │   │   └── /.env        # Binary environment per binary configuration
  │   ├── /...             # etc.
  ├── /pkg/                # Libraries
  │   ├── /auth/           # Authorization tools
  │   ├── /...             # etc.
  │   ├── /domain/         # Domain libraries
  │   │   ├── /user/       # User domain
  │   │   │   ├── /main.go # Main user domain entrypoint
  │   │   ├── /...         # etc.
  │   ├── /http/           # Http utils
  │   ├── /...             # etc.
  │   ├── /...             # More internal libraries
  ├── /vendor/             # Vendor libraries
  ├── docker-compose.yml   # Defines Docker services, networks and volumes per developer environment
  ├── Dockerfile           # Docker image for production
  ├── Makefile             # Commands for building a Docker image for production and deployment
  ├── .env                 # Project environment configuration
  └── bootstart.sh         # Configuration script for docker containers
*/
package goapiboilerplate
