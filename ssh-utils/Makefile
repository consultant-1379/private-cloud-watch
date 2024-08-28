PROJECT=github.com/erixzone/xaas/platform/ssh-utils
PACKAGES=$(PROJECT)/pkg/...
COMMANDS=$(PROJECT)/cmd/...
PLUGIN=$(PROJECT)/plugin

all: build

build:
	@echo "Building ssh-utils:"
	@go build $(PACKAGES) $(COMMANDS) $(PLUGIN)/...

install: build
	@echo "Installing ssh-utils:"
	@go install $(COMMANDS)
	@ls plugin | while read i; \
	do \
		go build -o $(GOPATH)/bin/ssh-$$i-plugin -buildmode=plugin $(PLUGIN)/$$i; \
	done
