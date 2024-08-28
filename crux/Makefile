PROJECT=github.com/erixzone/crux
VENDOR=$(PROJECT)/vendor
# there can be only one thing in $GOPATH; see checkgopath target below
GOPROJ=$(GOPATH)/src/$(PROJECT)

ifndef BUILDBASE
	BUILDBASE := $(PWD)
endif

BUILDDIR := $(BUILDBASE)/build
ARCHIVES := $(PWD)/vendor/archives
TESTS := $(PWD)/tests

export MAKEFILE := $(PWD)/Makefile
export PKG_CONFIG_PATH := $(BUILDDIR)/lib/pkgconfig
export PKG_CONFIG_LIBDIR := $(BUILDDIR)/lib/pkgconfig
export LD_LIBRARY_PATH := $(BUILDDIR)/lib
export LIBRARY_PATH := $(LD_LIBRARY_PATH)
export CDEPS_BIN := $(BUILDDIR)/bin

# Prefer our built binaries over anything that might be on the local system.
export PATH := $(CDEPS_BIN):$(PATH)

HOST ?= $(shell uname)
CPU ?= $(shell uname -m)
UID ?= $(shell id -u)
GID ?= $(shell id -g)
USER ?= $(shell whoami)
GROUP ?= $(shell id -gn)

# if no host is identifed (no uname tool)
# we assume a Linux-64bit build
ifeq ($(HOST),)
  HOST = Linux
endif

# identify CPU
ifeq ($(CPU), x86_64)
  HOST := $(HOST)64
else
ifeq ($(CPU), i686)
  HOST := $(HOST)32
endif
endif

# if you need to use proxy's for fetching ubuntu stuff, then uncomment the following, adjusting the IP:port.
# note that there may still be problems in various go commands
#PROXY := --build-arg=http_proxy=http://a.b.c.d:p --build-arg=https_proxy=https://e.f.g.h:q

#############################################
# now we find out the target OS for
# which we are going to compile in case
# the caller didn't yet define OS himself
ifndef (OS)
  ifeq ($(HOST), Linux64)
    arch = gcc
  else
  ifeq ($(HOST), Linux32)
    arch = 32
  else
  ifeq ($(HOST), Darwin64)
    arch = clang
  else
  ifeq ($(HOST), Darwin32)
    arch = clang
  endif
  endif
  endif
  endif
endif

# Determine where to store container targets, which must always
# be in a separate directory from the 'host' targets.
KERNEL ?= $(shell uname -s)
CONTAINER_BUILD_DIR=$(BUILDBASE)/build/Linux_$(CPU)

# Do a test to make sure we have an acceptable protoc locally. If this fails,
# cdeps will build the vendored protobuf. The logic exists in the "cdeps"
# target definition below.
PROTOC_NEED_AT_LEAST = 3.3
PROTOC_VERSION := $(shell protoc --version 2> /dev/null | sed 's/^.* //g')
PROTOC_VERSION_TEST := $(shell expr $(PROTOC_VERSION) \>= $(PROTOC_NEED_AT_LEAST) 2> /dev/null)


all: install

checkdeps:
	@echo "Checking deps:"
	@(env bash $(PWD)/tools/checkdeps.sh)

# this check used to be more complicated, but all the other uses of $GOPATH assume it is a single entry
checkgopath:
	@if [ "${GOPROJ}" != "${PWD}" ]; then\
		echo "Checking gopath: our pwd(${PWD}) is not "'$$GOPATH/src/$$PROJECT'"(${GOPROJ})"; exit 1;\
	fi

$(BUILDDIR):
	mkdir $(BUILDDIR)

protobuf=protobuf-3.3.0
myriad=myriad_d120df6_20190727151803Z

# the following go install's are to compensate for stringer not honoring vendoring correctly
deps: $(BUILDDIR) cdeps \
		$(BUILDDIR)/myriad/bin/myriad
	cd vendor/github.com/twmb/algoimpl/go/graph; go install

# Build protobuf if there's no available protoc, or if the available one didn't
# pass the protoc version test. If we have a good protoc, don't build anything.
ifneq ($(PROTOC_VERSION_TEST), 1)
cdeps: $(CDEPS_BIN)/protoc
else
cdeps: ;
endif

gmock=$(ARCHIVES)/gmock-1.7.0.zip
# HACK(skaar): Below if handles libtool issue on NixOS
$(CDEPS_BIN)/protoc: $(ARCHIVES)/$(protobuf).tar.gz $(gmock)
	cd $(BUILDDIR); tar zxvf $<; cd $(protobuf); \
	rm -fr gmock;\
	unzip -q $(gmock) -d gmock;\
	if [ -f /run/current-system/sw/bin/nix-env ]; then \
		echo "Building on NixOS"; \
		nix-env -f $(ARCHIVES)/protobufenv.nix -i env-protobuf;\
		$(HOME)/.nix-profile/bin/load-env-protobuf ./autogen.sh;\
	else \
		./autogen.sh;\
	fi;\
	./configure --prefix=$(BUILDDIR); \
	make; make install

$(BUILDDIR)/myriad/bin/myriad:	$(ARCHIVES)/$(myriad).tar.gz
	@tools/buildgodepfromarchive.sh $(BUILDDIR) "github.com/erixzone" "myriad" $(myriad) $<

# recipes to generate files with enumerator, lex, yacc

%_enum.go: %.enum
	$(GOPATH)/bin/enumerator <$< >$@

%.l.go: %.l
	golex $<

run-yacc = FLAGS=`sed -n '/^\/\/yacc:flags /s///p' $<` && \
	goyacc $$FLAGS -o $@ $<

%.y.go: %.y
	$(run-yacc)

# recipes for generated files that are dropped outside of the source directory
# GENDIR is evaluated when make is run there (see %.generate)

$(GENDIR)/%.l.go: %.l
	mkdir -p $(GENDIR)
	golex $< && mv $*.l.go $(GENDIR)

$(GENDIR)/%.y.go: %.y
	mkdir -p $(GENDIR)
	$(run-yacc)

# recipe to run make in a subdirectory
# GENDIR is set to a reasonable default and the top-level Makefile is inherited

%.generate:
	@export GENDIR=$(PWD)/gen/$$(basename $*); \
	MAKEFILES=$(MAKEFILE) $(MAKE) -C $*

gentools: deps pkg/alembicpb.generate
	cd tools/server && go install

GENDIRS=$(dir $(shell find $(PWD)/cmd $(PWD)/pkg -name Makefile))

GENLIST=$(addsuffix .generate,$(GENDIRS) protos)

$(GENLIST): gentools

generate: $(GENLIST)

genclean:
	rm -rf gen
	rm `find $(PWD)/cmd $(PWD)/pkg -name '*.[ly].go' -o -name '*_enum.go'`

gitconfig:
	@git config filter.tagger.clean ./tools/git-filters/tagger/clean.sh
	@git config filter.tagger.smudge ./tools/git-filters/tagger/smudge.sh

getdeps: checkdeps checkgopath
	@go install $(VENDOR)/github.com/kardianos/govendor && echo "Installed govendor:"
	@go install $(VENDOR)/github.com/golang/protobuf/protoc-gen-go && echo "Installed proto/protoc-gen-go:"
	@go install $(VENDOR)/golang.org/x/tools/cmd/goyacc && echo "Installed goyacc:"
	@go install $(VENDOR)/golang.org/x/tools/cmd/stringer && echo "Installed stringer:"
	@go install $(VENDOR)/github.com/golang/lint/golint && echo "Installed golint:"
	@go install $(VENDOR)/github.com/kivikakk/golex && echo "Installed golex:"
	@go install $(VENDOR)/github.com/hashicorp/go-rootcerts && echo "Installed rootcerts:"
	@go install $(VENDOR)/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway && echo "Installed grpc gateway generator"
	@install -m 755 tools/enumerator.sh $(GOPATH)/bin/enumerator && echo "Installed enumerator:"
	@install -m 755 tools/muster0.sh $(GOPATH)/bin/muster0 && echo "Installed muster0:"

ldflags:
	@go run ./tools/gen-ldflags.go ldflags


build: ldflags getdeps generate #testplugin
	go build $(PROJECT)/pkg/...
	go build $(PROJECT)/cmd/...
	go build -o $(GOPATH)/bin/pastiche-testserver  $(PROJECT)/pkg/pastiche/testserver
	go build -o $(GOPATH)/bin/pastiche-demo  $(PROJECT)/pkg/pastiche/serverdemo
	go build -o $(GOPATH)/bin/grpcsigsrv-test  $(PROJECT)/pkg/grpcsig/test/server
	go build -o $(GOPATH)/bin/grpcsigcli-test  $(PROJECT)/pkg/grpcsig/test/client
	go build -o $(GOPATH)/bin/registersrv-test $(PROJECT)/pkg/register/server
	go build -o $(GOPATH)/bin/registercli-test $(PROJECT)/pkg/register/client
	go build -o $(GOPATH)/bin/pubkeydb $(PROJECT)/pkg/grpcsig/pubkeydb



install: ldflags build verifiers
	@for i in example fulcrum; do go build -o $(GOPATH)/bin/plugin_$$i -buildmode=plugin $(PROJECT)/plugin/$$i; done
	@go install --ldflags "`cat ldflags`" $(GOFLAGS) $(PROJECT)/cmd/...

# to run a single test for debugging, use something like
#	go test -gocheck.vv=true -check.f KhanKhan github.com/erixzone/crux/pkg/khan...
# when in the pkg/khan directory

test: install
	# currently, autotest is broken due to logging interactions
	cd pkg/begat/lib; $(GOPATH)/bin/testgen begat --unit=true auto.spec > auto_test.go; rm auto_test.go
	ssh-agent go test $(PROJECT)/pkg/...
	go test $(PROJECT) $(PROJECT)/cmd/...

ftest:
	go test $(PROJECT)/tests/functional/...

# having container as a prereq is tedious, but you need an up to date plugin binary
ftestcc: $(BUILDDIR)/containers/crux-build.txt
	@docker run --user $(UID):$(GID) \
		-v $(BUILDBASE):/crux/src/github.com/erixzone/crux \
		-v $(CONTAINER_BUILD_DIR)/gobin:/crux/bin \
		-v $(CONTAINER_BUILD_DIR)/.cache:/.cache \
		erixzone/crux-build sh -c "make ftestc"
ftestc:
	go test $(PROJECT)/tests/functional-container/...

itest:
	@echo "integration tests:"
	@for i in $(TESTS)/integration/[a-zA-Z]*; do cd $$i; echo "	`basename $$i`:"; sh test.sh; done

tests:	test ftest itest ftestcc
	@echo "all tests done"

.PHONY: test tests

gorka: checkgopath
	cd $(GOPATH); rm -fr pkg/* bin/* gen $(BUILDDIR)

verifiers:
	@go vet $(PROJECT)/pkg/...
	@go vet $(PROJECT)/cmd/...
	@golint $(PROJECT)/pkg/...
	@golint $(PROJECT)/cmd/...

# Radix (c)package: this contains relevant binaries under bin/ (crux, glpsol, minio)
# and neccesary dynamic libraries under lib/
# Due to differences between linux distributions, we must build all libraries using a common
# 'ubuntu14.04' container image for the integration tests. Thus the 'cpackage' target builds
# these things in a container. On a non-Linux system, Docker is always used.
$(CONTAINER_BUILD_DIR)/package/crux.tar.gz: cpackage
	@echo "Building $@:"

# cpackage: always build this inside a container.
cpackage: $(BUILDDIR)/containers/crux-build.txt
	@echo "Building crux in a Docker container:"
	@mkdir -p $(CONTAINER_BUILD_DIR)/gobin
	@mkdir -p $(CONTAINER_BUILD_DIR)/gopkg
	@mkdir -p $(CONTAINER_BUILD_DIR)/.cache
	docker run --rm --user $(UID):$(GID) \
		-v $(BUILDBASE):/crux/src/github.com/erixzone/crux \
		-v $(CONTAINER_BUILD_DIR):/crux/src/github.com/erixzone/crux/build \
		-v $(CONTAINER_BUILD_DIR)/gobin:/crux/bin \
		-v $(CONTAINER_BUILD_DIR)/gopkg:/crux/pkg \
		-v $(CONTAINER_BUILD_DIR)/.cache:/.cache \
		erixzone/crux-build sh -c "make package"

# package: There are external dependencies (shed, macropod) that rely on this
# target working in a certain way. On Linux, just build directly. On Darwin,
# build a container.
ifeq ($(KERNEL), Linux)
package: install
	@echo "=====building package"
	@rm -fr build/package
	@mkdir -p build/package/bin build/package/lib
	@echo "== Building plugin's from $(PROJECT)/plugin"
	@for i in example fulcrum; do go build -o build/package/bin/plugin_$$i -buildmode=plugin $(PROJECT)/plugin/$$i; done
	@for i in crux pastiche fulcrum myriad ripstop organza muster0; do cp $(GOPATH)/bin/$$i build/package/bin; done
	#@sh -c "(cd $(LD_LIBRARY_PATH) ; find . -name '*\.so\.*' | tar cf - --files-from - ) | ( cd build/package/lib ; tar xf - )"
	@ls -l build/package/bin
	@sh -c "cd build/package; tar czf - --owner=0 --group=0 bin" > build/crux.tar.gz
	@mv build/crux.tar.gz build/package/
	@echo "==========package done"
else
package: cpackage
endif

.PHONY: package cpackage

# Docker containers are represented in the build tree for Make via these
# text files which are the output of `docker images -q <ImageName> > file.txt`
# This contains an abbreviated SHA1.
container: crux-main crux-build

# Container targets

crux-ubuntu: $(BUILDDIR)/containers/crux-ubuntu.txt
crux-build: $(BUILDDIR)/containers/crux-build.txt
crux-main: $(BUILDDIR)/containers/crux-main.txt

.PHONY: container crux-ubuntu crux-build crux-main

$(BUILDDIR)/containers/crux-build.txt: $(BUILDDIR)/containers/crux-ubuntu.txt
$(BUILDDIR)/containers/crux-main.txt: $(BUILDDIR)/containers/crux-ubuntu.txt

# N.B. if "docker build..." fails, rm (by repository) still leaves cruft behind

$(BUILDDIR)/containers/%.txt: $(CONTAINER_BUILD_DIR)/%.d
	@echo "Building $* container."
	mkdir -p $(BUILDDIR)/containers
	docker image rm -f erixzone/$* >/dev/null 2>&1 || true
	(cd $<; docker build -t erixzone/$* .)
	docker images -q erixzone/$* >$@

.PRECIOUS: $(CONTAINER_BUILD_DIR)/%.d

$(CONTAINER_BUILD_DIR)/%.d: $(BUILDBASE)/tools/%.Dockerfile
	rm -rf $@
	mkdir -p $@
	cp $< $@/Dockerfile

# Container: erixzone/crux-main
$(CONTAINER_BUILD_DIR)/crux-main.d: $(BUILDBASE)/tools/crux-main.Dockerfile $(CONTAINER_BUILD_DIR)/package/crux.tar.gz
	rm -rf $@
	mkdir -p $@
	sed -e 's/^ARG UID=/&$(UID)/; s/^ARG GID=/&$(GID)/' \
	    -e 's/^ARG USER=/&$(USER)/; s/^ARG GROUP=/&$(GROUP)/' \
	    $(BUILDBASE)/tools/crux-main.Dockerfile >$@/Dockerfile
	cp $(BUILDBASE)/tools/crux-main.syslog-ng.conf $@
	ln $(CONTAINER_BUILD_DIR)/package/crux.tar.gz $@
