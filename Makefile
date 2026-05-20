IMG ?= docker.io/lixd96/i-dra-driver:latest

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/i-dra-driver cmd/main.go

.PHONY: build-image
build-image:
	docker build -t ${IMG} .
