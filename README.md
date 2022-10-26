# Megalos CNI

This repository contains the Golang source code for the Megalos CNI Plugin and the Megalos BGP Manager.
Megalos CNI is compatible with Kubernetes **v1.25+**. Previous versions **are not supported**.

This plugin creates pure L2 LANs distributed across different worker nodes using VXLAN.

### If you want to use Kathara with Megalos Manager without any changes, you should only download the `kathara-daemonset.yml` file.

## Usage

Before using this DaemonSet, [**Multus CNI**](https://github.com/intel/multus-cni) must be deployed in the cluster.

After that you can deploy the Kathara DaemonSet using:
```
kubectl create -f kathara-daemonset.yml
```
**Beware**: Megalos CNI is used only for additional Pod interfaces created by Multus CNI! For the `eth0` interface (required by Kubernetes) you must leverage on another CNI that manages L3 (e.g. Flannel, Calico).

## How it works

This CNI creates a VXLAN network overlay over the Kubernetes cluster network. By default, VXLAN uses L3 multicast groups to deliver BUM traffic and to learn remote MAC Addresses, but multicast traffic is not always permitted inside a Kubernetes cluster network (see public clouds). To overcome this limitation, we disable the default VXLAN MAC Learning and we replace it with EVPN BGP, which ensures unicast traffic.

The BGP Manager works as follows:
- A DaemonSet is deployed on worker nodes, and each worker deploys a Pod that is a BGP speaker.
- A Service is created, with TCP port 179 and it is linked to a Deployment that contains a Pod that acts as a BGP Route Reflector. The Route Reflector accepts dynamic BGP peerings, so it does not require further configuration. 
- Each worker node Pod has a single BGP peering with the BGP Route Reflector (using the Service Cluster IP Address).

When a Pod is created, the CNI works as follows:
- Each additional interface of the Pod is assigned to a specific VNI. Each VNI is associated to a different L2 LAN. In this way, we can distribute L2 LANs inside the cluster.
- On the worker node where the Pod is deployed, VTEPs for the specific Pod VNIs are created (if not already present).
- Connections between Pod interfaces and their speficic VTEPs are created using `veth` pairs.

At this point, the BGP control plane is able to fetch the information of Pod network interfaces. So, MAC Addresses of Pod interfaces are announced over BGP to the Route Reflector and reflected to all the other worker BGP speakers. Each worker BGP Pod receives the announce: if a VTEP for a specific VNI announced in the BGP message is present, the VTEP saves the association between MAC Address and the IP Address of the worker node where the new interfaces of that VNI are deployed.

## Building from source

In this repository you'll find two folders:

- `megalos-cni`: Golang CNI source code for Megalos.
- `bgp-manager`: Dockerfile and Bash scripts to create the `kathara/megalos-bgp-manager` Docker Image.

### Steps to build and deploy a custom version of the CNI

1. Change the `IMAGE_NAME` variable in `Makefile` with a custom tag `<CUSTOM_NAME>`.
2. Run on terminal `make all`, this will:
    1. Create a Docker container that will build the CNI binary from source for both `amd64` and `arm64` architectures.
    2. Push the Docker Image on your Docker Hub Repository.
3. Change the `kathara-daemonset.yml` file and replace `kathara/megalos-bgp-manager` with `<CUSTOM_NAME>`.
4. Install the DaemonSet in your Kubernetes cluster using `kubeadm create -f kathara-daemonset.yml`.
