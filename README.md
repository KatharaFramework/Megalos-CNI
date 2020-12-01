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

This CNI creates a VXLAN network overlay over the Kubernetes cluster network. By default, VXLAN uses L3 multicast groups to deliver BUM traffic and to learn remote MAC Addresses, but multicast traffic is not always permitted inside a Kubernetes cluster network (see public clouds). To overcome this limitation, we disable the default VXLAN MAC Learning and we replace it with EVPN BGP, which ensures unicast traffic.

The BGP Manager works as follows:
- A DaemonSet is deployed on the master node, it contains a Pod that is a BGP speaker.
- A DaemonSet is deployed on worker nodes, and each worker deploys a Pod that is a BGP speaker.
- Each worker node Pod has a BGP peering with the BGP speaker Pod in the master node.
- The BGP speaker Pod in the master node acts as a BGP Route Reflector.

When a Pod is created, the CNI works as follows:
- Each additional interface of the Pod is assigned to a specific VNI. Each VNI is associated to a different L2 LAN. In this way, we can distribute L2 LANs inside the cluster.
- On the worker node where the Pod is deployed, VTEPs for the specific Pod VNIs are created (if not already present).
- Connections between Pod interfaces and their speficic VTEPs are created using `veth` pairs.

At this point, the BGP control plane is able to fetch the information of Pod network interfaces. So, MAC Addresses of Pod interfaces are announced over BGP to the master and reflected to all the workers. Each worker BGP speaker Pod receives the announce: if a VTEP for a specific VNI announced in the BGP message is present, the VTEP saves the association between MAC Address and the IP Address of the worker node where the new interfaces of that VNI are deployed.

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
