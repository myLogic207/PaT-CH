GO = go
OUT = ./out/patch
SRC = ./src/
DOCKER_USERNAME ?= myLogic207
APPLICATION_NAME ?= pat-ch
VERSION ?= 0.0.1

.PHONY: all compile build run clean

all: build

compile:
	echo "Compiling for every OS and Platform"
	pushd .
	cd ${SRC}
	GOOS=freebsd GOARCH=386 ${GO} build -o ${OUT}-freebsd-386 .
	GOOS=linux GOARCH=386 ${GO} build -o ${OUT}-linux-386 .
	GOOS=windows GOARCH=386 ${GO} build -o ${OUT}-windows-386 .
	popd

build: main.go go.mod go.sum
	pushd .
	cd ${SRC}
	${GO} build -o ${OUT} .
	popd

run:
	cd ${SRC}
	${GO} run .

docker:
	docker build --tag ${APPLICATION_NAME}:${VERSION} .

clean: 
	rm -f ${OUT}
