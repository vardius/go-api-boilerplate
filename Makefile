# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)

# GENERIC TASKS
all-%:
	for BIN in `ls cmd`; do
		@$(MAKE) --no-print-directory BIN=$(BIN) setup $*
	done
build-%:
	@$(MAKE) --no-print-directory BIN=$* setup build
run-%:
	@$(MAKE) --no-print-directory BIN=$* setup run
stop-%:
	@$(MAKE) --no-print-directory BIN=$* setup stop
rm-%:
	@$(MAKE) --no-print-directory BIN=$* setup rm
release-%:
	@$(MAKE) --no-print-directory BIN=$* setup release
publish-%:
	@$(MAKE) --no-print-directory BIN=$* setup publish
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

# CONFIG TASK
# import config
setup:
	configfile=.$(BIN).env
	include $(configfile)
	export $(shell sed 's/=.*//' $(configfile))

# HTTPS TASK
# Generate key
key:
	openssl genrsa -out server.key 2048
	openssl ecparam -genkey -name secp384r1 -out server.key

# Generate self signed certificate
cert:
	openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650

# DOCKER TASKS
# Build the container
build:
	docker build --no-cache --build-arg BIN=$(BIN) -t $(BIN) .

# Run container on port configured in `.env`
run:
	docker run -i -t --rm --env-file=./.env -p=$(PORT):$(PORT) --name="$(BIN)" $(BIN))

stop:
	docker stop $(BIN)

rm: stop
	docker rm $(BIN)

# Docker release - build, tag and push the container
release: build publish

# Docker publish
publish: repo-login publish-latest publish-version

publish-latest: tag-latest
	@echo 'publish latest to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):latest

publish-version: tag-version
	@echo 'publish $(VERSION) to $(REGISTRY)'
	docker push $(REGISTRY)/$(BIN):$(VERSION)

# Docker tagging
tag: tag-latest tag-version

tag-latest:
	@echo 'create tag latest'
	docker tag $(BIN) $(REGISTRY)/$(BIN):latest

tag-version:
	@echo 'create tag $(VERSION)'
	docker tag $(BIN) $(REGISTRY)/$(BIN):$(VERSION)

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
repo-login:
	@eval $(CMD_REPOLOGIN)

# output to version
version:
	@echo $(VERSION)
