#!/usr/bin/make -f

.PHONY: gobuild

gobuild:
	go mod download
	GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build src/megalos.go src/common_utils.go src/vxlan_utils.go src/veth_utils.go
	chmod 777 ./megalos
	