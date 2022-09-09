#!/usr/bin/make -f

BUILDX=docker buildx build --platform linux/amd64,linux/arm64
IMAGE_NAME=kathara/megalos-bgp-manager

.PHONY: clean create-builder delete-builder all image

all: image

clean: delete-builder
	rm -f ./bgp-manager/cni-bin/megalos*
	rm -f ./megalos-cni/megalos

gobuild_docker_%:
	docker run -ti --rm -v `pwd`/megalos-cni/:/root/go-src golang:alpine3.14 /bin/sh -c "apk add -U make && cd /root/go-src && make gobuild_$*"

image: clean gobuild_docker_amd64 gobuild_docker_arm64 create-builder
	mv ./megalos-cni/megalos-amd64 ./bgp-manager/cni-bin/megalos-amd64
	mv ./megalos-cni/megalos-arm64 ./bgp-manager/cni-bin/megalos-arm64
	$(BUILDX) -t ${IMAGE_NAME} --push ./bgp-manager

create-builder:
	docker buildx create --name meg-cni-builder --use
	docker buildx inspect --bootstrap

delete-builder:
	docker buildx rm meg-cni-builder || true