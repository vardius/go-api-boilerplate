# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)

# HELP
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

version: ## Show version
	@echo $(VERSION)

# HTTPS TASK
key: ## [HTTP] Generate key
	openssl genrsa -out server.key 2048
	openssl ecparam -genkey -name secp384r1 -out server.key

cert: ## [HTTP] Generate self signed certificate
	openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650

# DOCKER TASKS
docker-build: ## [DOCKER] Build given container. Example: `make docker-build BIN=user`
	docker build -f cmd/$(BIN)/Dockerfile --no-cache --build-arg BIN=$(BIN) -t $(BIN) .

docker-run: ## [DOCKER] Run container on given port. Example: `make docker-run BIN=user PORT=3000`
	docker run -i -t --rm -p=$(PORT):$(PORT) --name="$(BIN)" $(BIN)

docker-stop: ## [DOCKER] Stop docker container. Example: `make docker-stop BIN=user`
	docker stop $(BIN)

docker-rm: docker-stop ## [DOCKER] Stop and then remove docker container. Example: `make docker-rm BIN=user`
	docker rm $(BIN)

docker-publish: aws-repo-login docker-publish-latest docker-publish-version ## [DOCKER] Docker publish. Example: `make docker-publish BIN=user REGISTRY=https://your-registry.com`

docker-publish-latest: tag-latest
	@echo 'publish latest to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):latest

docker-publish-version: tag-version
	@echo 'publish $(VERSION) to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):$(VERSION)

docker-tag: docker-tag-latest docker-tag-version ## [DOCKER] Tag current container. Example: `make docker-tag BIN=user REGISTRY=https://your-registry.com`

docker-tag-latest:
	@echo 'create tag latest'
	docker tag $(BIN) $(REGISTRY)/$(BIN):latest
	docker tag $(BIN) $(REGISTRY)/$(BIN):latest

docker-tag-version:
	@echo 'create tag $(VERSION)'
	docker tag $(BIN) $(REGISTRY)/$(BIN):$(VERSION)

docker-release: docker-build docker-publish ## [DOCKER] Docker release - build, tag and push the container. Example: `make docker-release BIN=user REGISTRY=https://your-registry.com`

# # KUBERNETES TASKS
# kubernetes-create: ## [KUBERNETES] Create kubernetes deployment. Example: `make kubernetes-create BIN=user`
# 	kubectl create -f cmd/$(BIN)/deployment.yml

# kubernetes-replace: ## [KUBERNETES] Replace kubernetes deployment. Example: `make kubernetes-replace BIN=user`
# 	kubectl replace --force -f cmd/$(BIN)/deployment.yml

# HELM TASKS
helm-install: ## [HELM] Deploy the Helm chart. Example: `make helm-install BIN=user`
	helm install -n $(BIN) cmd/$(BIN)/helm-chart/

helm-upgrade: ## [HELM] Update the Helm chart. Example: `make helm-upgrade BIN=user`
	helm upgrade $(BIN) cmd/$(BIN)/helm-chart/

helm-history: ## [HELM] See what revisions have been made to the chart. Example: `make helm-history BIN=user`
	helm history $(BIN)

# TELEPRESENCE TASKS
telepresence-swap-local: ## [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local process. Example: `make telepresence-swap-local BIN=user PORT=3000`
	go build -o cmd/$(BIN)/$(BIN) cmd/$(BIN)/main.go
	telepresence \
	--swap-deployment $(BIN) \
	--expose 3000 \
	--run ./cmd/$(BIN)/$(BIN) \
	--port=$(PORT) \
	--method vpn-tcp

telepresence-swap-docker: ## [TELEPRESENCE] Replace the existing deployment with the Telepresence proxy for local docker image. Example: `make telepresence-swap-docker BIN=user PORT=3000`
	telepresence \
	--swap-deployment $(BIN) \
	--docker-run -i -t --rm -p=$(PORT):$(PORT) --name="$(BIN)" $(BIN):latest

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

aws-repo-login: ## [HELPER] login to AWS-ECR
	@eval $(CMD_REPOLOGIN)
