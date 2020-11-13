# Megalos CNI

This repository contains the Golang source code for the Megalos CNI Plugin and the Megalos BGP Manager (written in Python).

This plugin creates pure L2 LANs distributed across different worker nodes using VXLAN.

### If you want to use Kathara with Megalos Manager without any changes, you should only download the `kathara-daemonset.yml` file.

## Usage

Before using this DaemonSet, [**Multus CNI**](https://github.com/intel/multus-cni) must be deployed in the cluster.

After that you can deploy the Kathara DaemonSet using:

`kubectl create -f kathara-daemonset.yml`

**Beware**: Megalos CNI is used only for additional Pod interfaces created by Multus CNI! For the `eth0` interface (required by Kubernetes) you must leverage on another CNI that manages L3 (e.g. Flannel, Calico...).

## How it works

This CNI creates a VXLAN network overlay over the Kubernetes cluster network.
The default behaviour of VXLAN is to use multicast groups to deliver BUM traffic, but not on any Kubernetes cluster network multicast traffic is permitted.
To avoid the usage of multicast IP addresses, we use EVPN-BGP in such way:
- We deploy on the master node a BGP speaker.
- We deploy on each worker node a `kube-system` Pod with a BGP speaker.
- Each worker node has a BGP peering with the BGP speaker in the master node.
- The BGP speaker in the master node acts as a BGP Route Reflector.
With this setup, when a Pod is started, the MAC Addresses of each of its network interfaces are announced over BGP to the master and reflected to all the workers, so each VTEP knows the association between each MAC Address and the IP Address of the worker node where it is deployed.
So all the traffic over the Kubernetes cluster network is unicast.

## Building from source

In this repository you'll find two folders:

- `megalos-cni`: Golang CNI source code for Megalos.
- `bgp-manager`: Dockerfile and Python scripts to create the `kathara/megalos-bgp-manager` Docker Image.

### Steps to build and deploy a custom version of the CNI

1. Change the `IMAGE_NAME` variable in `Makefile` with a custom tag `<CUSTOM_NAME>`.
2. Run on terminal `make all` (Golang should be installed, all dependencies are automatically resolved).
3. Push the Docker Image on your Docker Hub Repository using `docker push <CUSTOM_NAME>`.
4. Change the `kathara-daemonset.yml` file and replace `kathara/megalos-bgp-manager` with `<CUSTOM_NAME>`.
5. Install the DaemonSet in your Kubernetes cluster using `kubeadm create -f kathara-daemonset.yml`.
