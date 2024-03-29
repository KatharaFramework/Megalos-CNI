# Megalos CNI

This repository contains the Golang source code for the Megalos CNI Plugin and the Megalos BGP Manager.
Megalos CNI is compatible with Kubernetes **v1.25+**. Previous versions **are not supported**.

This plugin creates pure L2 LANs distributed across different worker nodes using VXLAN.

## Prerequisites 

Before using the `kathara-daemonset`, the [**Multus CNI**](https://github.com/intel/multus-cni) must be deployed in the cluster.

See the [official installation guide](https://github.com/k8snetworkplumbingwg/multus-cni/blob/master/docs/quickstart.md). 

## Usage

Once you have deployed the [**Multus CNI**](https://github.com/intel/multus-cni), you can deploy the `kathara-daemonset` simply typing:
```bash
kubectl apply -f https://raw.githubusercontent.com/KatharaFramework/Megalos-CNI/master/kathara-daemonset.yml
```
**Beware**: Megalos CNI is used only for additional Pod interfaces created by Multus CNI! For the `eth0` interface (required by Kubernetes) you must leverage on another CNI that manages L3 (e.g. [Flannel](https://github.com/flannel-io/flannel), [Calico](https://www.tigera.io/project-calico/)).

**NOTE**: currently, Megalos CNI does not work on Amazon Elastic Kubernetes Service (EKS), Azure Kubernetes Service (AKS) and ready-to-go cloud instances of Kubernetes, it only works on self-hosted clusters (since `iptables` needs to install some custom rules).

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

## Use Megalos CNI without Kathará

It is possible to leverage on Megalos CNI as a standalone Multus CNI, in order to create pure L2 networks.

After following the [Usage](https://github.com/KatharaFramework/Megalos-CNI#usage) section, we can deploy an example network contained in a `network.yaml` file:
```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: test-network-1
spec:
  config: '{
              "cniVersion": "0.3.0",
              "name": "net1",
              "type": "megalos",
              "suffix": "ntwork",
              "vxlanId": 10
            }'
```

The `config` JSON contains the CNI configuration. The values are:
- `name`: a name for the network (maximum 4 characters)
- `suffix`: used by Kathará to distinguish between networks with the same `name` of different users (6 characters)
- `vxlanId`: Megalos CNI relies on VXLAN, so you should assign a VXLAN ID manually (Kathará does it automatically)

At this point, we can create the network with the following command:
```bash
kubectl apply -f network.yaml
```

This configuration will create a VTEP, with VXLAN ID = 10, associated to the network `net1-ntwork`.
