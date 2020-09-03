export PROJECT ?= quarks-secret
export QUARKS_UTILS ?= tools/quarks-utils
export GROUP_VERSIONS ?= quarkssecret:v1alpha1

test-unit: tools
	$(QUARKS_UTILS)/bin/test-unit

test-cluster: tools
	bin/build-image
	$(QUARKS_UTILS)/bin/test-integration
	$(QUARKS_UTILS)/bin/test-cli-e2e

lint: tools
	$(QUARKS_UTILS)/bin/lint

build-image: tools
	bin/build-image

.PHONY: tools
tools:
	bin/tools

############ GENERATE TARGETS ############

generate: gen-kube

gen-kube: tools
	$(QUARKS_UTILS)/bin/gen-kube

gen-command-docs:
	rm -f docs/commands/*
	go run cmd/docs/gen-command-docs.go

gen-fakes:
	bin/gen-fakes

############ COVERAGE TARGETS ############

coverage: tools
	$(QUARKS_UTILS)/bin/coverage
