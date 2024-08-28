# Copyright 2018 The Prometheus Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Needs to be defined before including Makefile.common to auto-generate targets
DOCKER_ARCHS ?= amd64 armv7 arm64

include Makefile.common

DOCKER_IMAGE_NAME       ?= prometheus

.PHONY: assets
assets:
	@echo ">> writing assets"
	cd $(PREFIX)/web/ui && GO111MODULE=$(GO111MODULE) $(GO) generate -x -v $(GOOPTS)
	@$(GOFMT) -w ./web/ui

.PHONY: check_assets
check_assets: assets
	@echo ">> checking that assets are up-to-date"
	@if ! (cd $(PREFIX)/web/ui && git diff --exit-code); then \
		echo "Run 'make assets' and commit the changes to fix the error."; \
		exit 1; \
	fi

.PHONY: install
install:
	mkdir -p $(GOPATH)/bin
	make build && cp -a prometheus promtool $(GOPATH)/bin

.PHONY: container
container: prom-run.txt

UID ?= $(shell id -u)
GID ?= $(shell id -g)
USER ?= $(shell whoami)
GROUP ?= $(shell id -gn)

GO_VERSION=1.12.4
GO_TAR=go$(GO_VERSION).linux-amd64.tar.gz
GO_URL=https://dl.google.com/go/$(GO_TAR)

$(GO_TAR):
	curl --silent --show-error --output $@ ${GO_URL}

prom-build.txt: prom-build.Dockerfile $(GO_TAR)
	rm -f $@
	docker image rm erixzone/prom-build >/dev/null 2>&1 || true
	rm -rf prom-build-container.d
	mkdir prom-build-container.d
	cp ./prom-build.Dockerfile prom-build-container.d/Dockerfile
	ln $(GO_TAR) prom-build-container.d
	(cd prom-build-container.d; \
	 docker build -t erixzone/prom-build --build-arg=GO_TAR=$(GO_TAR) \
		--build-arg=UID=$(UID) --build-arg=GID=$(GID) \
		--build-arg=USER=$(USER) --build-arg=GROUP=$(GROUP) .)
	docker images -q erixzone/prom-build >$@

prom-src.tar.gz:
	git archive --format=tgz --prefix=prometheus/prometheus/ --output=$@ HEAD ./

prom-bin.tar.gz: prom-build.txt prom-src.tar.gz
	docker run --rm -v $$(pwd):/src erixzone/prom-build \
		sh -c "tar xzf /src/prom-src.tar.gz && \
			cd prometheus/prometheus && \
			make build && \
			tar czf /src/$@ prometheus promtool"

prom-run.txt: prom-run.Dockerfile prom-bin.tar.gz
	rm -f $@
	docker image rm erixzone/prom-run >/dev/null 2>&1 || true
	rm -rf prom-run-container.d
	mkdir prom-run-container.d
	cp ./prom-run.Dockerfile prom-run-container.d/Dockerfile
	cp ./exec-prometheus.sh prom-run-container.d
	ln prom-bin.tar.gz prom-run-container.d
	(cd prom-run-container.d; \
	 docker build -t erixzone/prom-run \
		--build-arg=UID=$(UID) --build-arg=GID=$(GID) \
		--build-arg=USER=$(USER) --build-arg=GROUP=$(GROUP) .)
	docker images -q erixzone/prom-run >$@

.PHONY: container-clean
container-clean:
	rm -f prom-src.tar.gz prom-bin.tar.gz
	rm -rf prom-build.txt prom-build-container.d
	rm -rf prom-run.txt prom-run-container.d
