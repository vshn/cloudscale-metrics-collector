# Commodore takes the root dir name as the component name
COMPONENT_NAME ?= cloudscale-metrics-collector
COMPONENT_SUBDIR ?= $(shell basename ${PWD})

compiled_path   ?= compiled/$(COMPONENT_NAME)/$(COMPONENT_NAME)
root_volume     ?= -v "$${PWD}/../:/$(COMPONENT_NAME)"
compiled_volume ?= -v "$${PWD}/$(compiled_path):/$(COMPONENT_NAME)"
commodore_args  ?= --search-paths . -n $(COMPONENT_NAME) --alias $(instance)

ifneq "$(shell which docker 2>/dev/null)" ""
	DOCKER_CMD    ?= $(shell which docker)
	DOCKER_USERNS ?= ""
else
	DOCKER_CMD    ?= podman
	DOCKER_USERNS ?= keep-id
endif
DOCKER_ARGS ?= run --rm -u "$$(id -u):$$(id -g)" --userns=$(DOCKER_USERNS) -w /$(COMPONENT_NAME)/$(COMPONENT_SUBDIR) -e HOME="/$(COMPONENT_NAME)"

JSONNET_FILES   ?= $(shell find . -type f -not -path './vendor/*' \( -name '*.*jsonnet' -or -name '*.libsonnet' \))
JSONNETFMT_ARGS ?= --in-place --pad-arrays
JSONNET_IMAGE   ?= docker.io/bitnami/jsonnet:latest
JSONNET_DOCKER  ?= $(DOCKER_CMD) $(DOCKER_ARGS) $(root_volume) --entrypoint=jsonnetfmt $(JSONNET_IMAGE)

YAMLLINT_ARGS   ?= --no-warnings
YAMLLINT_CONFIG ?= .yamllint.yml
YAMLLINT_IMAGE  ?= docker.io/cytopia/yamllint:latest
YAMLLINT_DOCKER ?= $(DOCKER_CMD) $(DOCKER_ARGS) $(root_volume) $(YAMLLINT_IMAGE)

VALE_CMD  ?= $(DOCKER_CMD) $(DOCKER_ARGS) $(root_volume) --volume "$${PWD}"/docs/modules:/pages docker.io/vshn/vale:2.1.1
VALE_ARGS ?= --minAlertLevel=error --config=/pages/ROOT/pages/.vale.ini /pages

ANTORA_PREVIEW_CMD ?= $(DOCKER_CMD) run --rm --publish 35729:35729 --publish 2020:2020 --volume "${PWD}/.git":/preview/antora/.git --volume "${PWD}/docs":/preview/antora/docs docker.io/vshn/antora-preview:3.0.1.1 --style=syn --antora=docs


COMMODORE_CMD  ?= $(DOCKER_CMD) $(DOCKER_ARGS) $(root_volume) docker.io/projectsyn/commodore:latest
COMPILE_CMD    ?= $(COMMODORE_CMD) component compile . $(commodore_args)
JB_CMD         ?= $(DOCKER_CMD) $(DOCKER_ARGS) --entrypoint /usr/local/bin/jb docker.io/projectsyn/commodore:latest install
GOLDEN_FILES    ?= $(shell find tests/golden/$(instance) -type f)

KUBENT_FILES    ?= $(shell echo "$(GOLDEN_FILES)" | sed 's/ /,/g')
KUBENT_ARGS     ?= -c=false --helm2=false --helm3=false -e
# Use our own kubent image until the upstream image is available
KUBENT_IMAGE    ?= docker.io/projectsyn/kubent:latest
KUBENT_DOCKER   ?= $(DOCKER_CMD) $(DOCKER_ARGS) $(root_volume) --entrypoint=/app/kubent $(KUBENT_IMAGE)

instance ?= cloudscale-metrics-collector
test_instances = tests/cloudscale-metrics-collector.yml tests/collector-exoscale-ch-gva-2-0.yml
