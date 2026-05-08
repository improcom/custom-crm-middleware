
.PHONY: default dev clean tidy build build-dev schema schema-gen

# To set local developer overrides, create Makefile.local
# To run something before build, set `PREBUILD` (like: `PREBUILD := schema-gen`)
# To change build service name, define `SERVICENAME` (like: `SERVICENAME := my-middleware`)
-include Makefile.local

# Affects the output binary name and all runtime identifiers: state directory, database file, and CLI usage.
SERVICENAME ?= crm-middleware

SRVNAME_FLAG := -X 'crm-middleware/build.serviceName=$(SERVICENAME)'
VERSION_FLAG := -X 'crm-middleware/build.version=v$(shell cat VERSION 2>/dev/null).$(shell date +%y%m%d)-$(shell git rev-parse --short=10 HEAD 2>/dev/null)'

PREBUILD ?=

default: clean tidy build

dev: clean tidy run-dev

fresh: clean
	rm -rf .$(SERVICENAME)

clean:
	rm -rf bin

tidy:
	go mod tidy

build: $(PREBUILD)
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w $(SRVNAME_FLAG) $(VERSION_FLAG)" -o bin/$(SERVICENAME)

run-dev: $(PREBUILD)
	CGO_ENABLED=0 go run -trimpath -ldflags "$(SRVNAME_FLAG) $(VERSION_FLAG)" -tags dev main.go

schema:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/schema ./tools/schema

schema-gen: schema
	bin/schema generate apps/salesforce/models apps/salesforce/schemas
