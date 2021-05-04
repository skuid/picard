REPO=picard
TAG ?= latest
BUILD_TAG ?= latest
SHA = $(shell git rev-parse --short HEAD)
REV = $(shell git rev-list --tags --max-count=1)
VERSION = $(shell git describe --tags $(REV))
WD = $(shell basename $(dir $(abspath $(dir $$PWD))))


test:
	# @make service
	@go test -cover ./...
	
testv:
	@go test -v -cover ./...
	
build:
	@go build -o picard