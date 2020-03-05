#!/usr/bin/make -f

MEGALOS_CNI_PATH=./megalos-cni
BGP_MANAGER_PATH=./bgp-manager
IMAGE_NAME=kathara/megalos-bgp-manager

.PHONY: all clean gobuild image

all: image

clean:
	rm -f ${BGP_MANAGER_PATH}/cni-bin/megalos
	rm -f ./megalos

gobuild:
	go get -v github.com/docker/libnetwork github.com/vishvananda/netlink github.com/docker/go-plugins-helpers/network github.com/google/uuid
	go get -v github.com/containernetworking/cni || true
	go get -v github.com/containernetworking/plugins || true
	go build ${MEGALOS_CNI_PATH}/megalos.go ${MEGALOS_CNI_PATH}/common_utils.go ${MEGALOS_CNI_PATH}/iptables_utils.go ${MEGALOS_CNI_PATH}/vxlan_utils.go ${MEGALOS_CNI_PATH}/veth_utils.go

image: clean gobuild
	mv ./megalos ${BGP_MANAGER_PATH}/cni-bin/megalos
	docker build -t ${IMAGE_NAME} ${BGP_MANAGER_PATH}
