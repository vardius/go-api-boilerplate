# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)

# import config.
# You can change the default config with `make cnf="config_special.env" build`
cnf ?= .env
include $(cnf)
export $(shell sed 's/=.*//' $(cnf))

# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

# HTTPS TASK
# Generate key
key:
	openssl genrsa -out server.key 2048
	openssl ecparam -genkey -name secp384r1 -out server.key

# Generate self signed certificate
cert:
	openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650

# CONFIG TASK
# import enviroments for binary
setup:
	@echo 'setup package .env file for $(BIN)'
	localCnf ?= ./cmd/$(BIN)/.env
	include $(localCnf)
	export $(shell sed 's/=.*//' $(localCnf))

# GENERIC TASKS
all-%:
	for BIN in $(shell ls cmd); do $(MAKE) --no-print-directory BIN=$$BIN setup $*; done
docker-build-%:
	@$(MAKE) --no-print-directory BIN=$* setup docker-build
docker-run-%:
	@$(MAKE) --no-print-directory BIN=$* setup docker-run
docker-stop-%:
	@$(MAKE) --no-print-directory BIN=$* setup docker-stop
docker-rm-%:
	@$(MAKE) --no-print-directory BIN=$* setup docker-rm
publish-latest-%:
	@$(MAKE) --no-print-directory BIN=$* setup publish-latest
publish-version-%:
	@$(MAKE) --no-print-directory BIN=$* setup publish-version
tag-%:
	@$(MAKE) --no-print-directory BIN=$* setup tag
tag-latest-%:
	@$(MAKE) --no-print-directory BIN=$* setup tag-latest
tag-version-%:
	@$(MAKE) --no-print-directory BIN=$* setup tag-version
release-%:
	@$(MAKE) --no-print-directory BIN=$* setup release
publish-%:
	@$(MAKE) --no-print-directory BIN=$* setup publish

# DOCKER TASKS
# Build the container
docker-build:
	docker build -f docker/cmd/Dockerfile --no-cache --build-arg BIN=$(BIN) PKG=$(PKG) -t local/$(BIN) .

# Run container on port configured in `.env`
docker-run:
	docker run -i -t --rm --env-file=./cmd/$(BIN)/.env -p=$(PORT):$(PORT) --name="$(BIN)" local/$(BIN)

docker-stop:
	docker stop $(BIN)

docker-rm: stop
	docker rm $(BIN)

# Docker publish
docker-publish: aws-repo-login docker-publish-latest docker-publish-version

docker-publish-latest: tag-latest
	@echo 'publish latest to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):latest

docker-publish-version: tag-version
	@echo 'publish $(VERSION) to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):$(VERSION)

# Docker tagging
docker-tag: docker-tag-latest docker-tag-version

docker-tag-latest:
	@echo 'create tag latest'
	docker tag $(BIN) $(REGISTRY)/$(BIN):latest

docker-tag-version:
	@echo 'create tag $(VERSION)'
	docker tag $(BIN) $(REGISTRY)/$(BIN):$(VERSION)

# Docker release - build, tag and push the container
docker-release: build publish

# KUBERNETES TASKS
kubernetes-create:
	kubectl create -f ./kubernetes/$(BIN)-deployment.yml

# HELPERS
# generate script to login to aws docker repo
CMD_REPOLOGIN := "eval $$\( aws ecr"
ifdef AWS_CLI_PROFILE
CMD_REPOLOGIN += " --profile $(AWS_CLI_PROFILE)"
endif
ifdef AWS_CLI_REGION
CMD_REPOLOGIN += " --region $(AWS_CLI_REGION)"
endif
CMD_REPOLOGIN += " get-login \)"

# login to AWS-ECR
aws-repo-login:
	@eval $(CMD_REPOLOGIN)

# output to version
version:
	@echo $(VERSION)
