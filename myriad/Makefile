PROJECT=github.com/erixzone/myriad
PACKAGES=$(PROJECT)/pkg/...
COMMANDS=$(PROJECT) $(PROJECT)/cmd/...

all: install

checkdeps:
	@echo "Checking deps:"
	@(env bash $(PWD)/tools/checkdeps.sh)

checkgopath:
	@echo "Checking if project is at $(GOPATH)"
	@for mcpath in $$(echo $(GOPATH) | sed 's/:/\n/g') ''; do [ "$$mcpath" -a -d $$mcpath/src/$(PROJECT) ] && break; done; [ "$$mcpath" ] && exit 0; echo "Project not found in $(GOPATH), please follow instructions provided in README.md" && exit 1

getdeps: checkdeps checkgopath
	@GO15VENDOREXPERIMENT=1 go install $(PROJECT)/vendor/github.com/kardianos/govendor && echo "Installed github.com/kardianos/govendor"

gitconfig:
	@git config filter.tagger.clean ./tools/git-filters/tagger/clean.sh
	@git config filter.tagger.smudge ./tools/git-filters/tagger/smudge.sh

verifiers: getdeps fmt

fmt:
	@GO15VENDOREXPERIMENT=1 go fmt $(PACKAGES) $(COMMANDS)

ldflags:
	@GO15VENDOREXPERIMENT=1 go run ./tools/gen-ldflags.go ldflags

build: getdeps verifiers ldflags
	@echo "Building myriad:"
	@GO15VENDOREXPERIMENT=1 go build --ldflags "`cat ldflags`" $(GOFLAGS) $(PACKAGES) $(COMMANDS)

test: build
	@./tools/vendor-test.sh
	@GO15VENDOREXPERIMENT=1 go test $(GOFLAGS) $(PACKAGES) $(COMMANDS)

gomake-all: build
	@GO15VENDOREXPERIMENT=1 go install --ldflags "`cat ldflags`" $(GOFLAGS) $(COMMANDS)

install: gomake-all

cover:
	export GO15VENDOREXPERIMENT=1; \
	for pkg in $$(go list ./pkg/...); \
	do cpath=`echo $$pkg | sed -e s@github.com/erixzone/myriad/@@`; \
	   go test -coverprofile ./$${cpath}/cover.out $${pkg}; \
	done

generate:
	@GO15VENDOREXPERIMENT=1 go generate $(PACKAGES)

clean:
	@find . -name cover.out -exec rm {} \;

check:
	@go vet $(PACKAGES) $(COMMANDS)
	@golint ./... | grep -v "vendor\|fake\|Fake\|pb.go" || true

export-list:
	@echo myriad myriadca

TS=$(shell git log -1 --pretty=format:"%ct" )
TS1=$(shell gawk -v TS=$(TS) 'BEGIN { print strftime("%Y%m%d%H%M%SZ", TS, "utc") }'  < /dev/null )
N=$(shell git log -1 --pretty=format:"myriad_%h_${TS1}" )
export:
	@git archive --format=tgz --prefix=$(N)/ --output=$(N).tar.gz `git log  -1 --pretty=format:"%H"`
	@ls -l $(N).tar.gz

.PHONY: clean test cover ldflags int-test-pouches export-list
