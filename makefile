GO = go
OUT = out
DOCKER_USERNAME ?= myLogic207
APP_NAME ?= pat-ch
VERSION ?= 0.0.1

.PHONY: all
all: build run

run: clean build redis-server-running
	./${OUT}/${APP_NAME}

.PHONY: compile
.ONESHELL:
compile:
	@echo "Compiling for every OS and Platform"
	mkdir -p ${OUT}
	GOOS=freebsd GOARCH=386 ${GO} build -o ../${OUT}/${APP_NAME}_freebsd-386 .
	GOOS=linux GOARCH=386 ${GO} build -o ../${OUT}/${APP_NAME}_linux-386 .
	GOOS=windows GOARCH=386 ${GO} build -o ../${OUT}/${APP_NAME}_windows-386 .

.PHONY: build
.ONESHELL:
SHELL := /bin/bash
build:
	mkdir -p ${OUT}
	${GO} build -o ../${OUT}/${APP_NAME} .

.PHONY: docker
docker:
	docker build --tag ${APPLICATION_NAME}:${VERSION} .

.PHONY: clean
clean: 
	rm -rf ${OUT}/

.PHONY: redis-server-running
redis-server-running:
	redis-server --daemonize yes --port 6379 --bind
