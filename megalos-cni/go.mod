module github.com/KatharaFramework/Megalos-CNI

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1

go 1.16

require (
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.1.1
	github.com/docker/libnetwork v0.8.0-dev.2.0.20210525090646-64b7a4574d14
	github.com/google/uuid v1.3.0
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
)
