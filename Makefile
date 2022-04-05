#!/usr/bin/make -f

IMAGE_NAME=kathara/megalos-bgp-manager

.PHONY: all clean gobuild image

all: image

clean:
	rm -f ./bgp-manager/cni-bin/megalos
	rm -f ./megalos-cni/megalos

gobuild_docker:
	docker run -ti --rm -v `pwd`/megalos-cni/:/root/go-src golang:alpine3.14 /bin/sh -c "apk add -U make && cd /root/go-src && make gobuild"

image: clean gobuild_docker
	mv ./megalos-cni/megalos ./bgp-manager/cni-bin/megalos
	docker build -t ${IMAGE_NAME} ./bgp-manager
