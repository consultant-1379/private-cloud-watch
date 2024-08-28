PROJECT=github.com/erixzone/xaas/platform/enm-vis
PACKAGES=$(PROJECT)/pkg/...
COMMANDS=$(PROJECT)/cmd/...

all: build

build:
	@echo "Building enm-vis:"
	@##go build $(PACKAGES)
	@go build $(COMMANDS)

install: build
	@echo "Installing enm-vis:"
	@go install $(COMMANDS)
