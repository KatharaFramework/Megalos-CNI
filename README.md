# Megalos CNI

This repository contains the Golang source code for the Megalos CNI Plugin and the Megalos BGP Manager (written in Python).

This plugin creates pure L2 LANs using VXLAN, VXLAN companion bridges and veths.

### If you want to use Kathara on Kubernetes without any changes, you should only download the `kathara-daemonset.yml` file.

## Usage

Before using this DaemonSet, [**Multus CNI**](https://github.com/intel/multus-cni) must be started in the cluster.

After that you can start the Kathara DaemonSet using:

`kubeadm create -f kathara-daemonset.yml`

## Building from source

In this repository you'll find two folders:

- `megalos-cni`: CNI source code for Megalos.
- `bgp-manager`: Dockerfile and Python scripts to create the `kathara/megalos-bgp-manager` Docker Image.

### Steps

Type on terminal 

1. Change the `IMAGE_NAME` variable in `Makefile` with a custom tag `<CUSTOM_NAME>`
2. Run on terminal `make all` (Golang should be installed, all dependencies are automatically resolved)
3. Push the Docker Image on your Docker Hub Repository using `docker push <CUSTOM_NAME>`.
4. Change the `kathara-daemonset.yml` file and replace `kathara/megalos-bgp-manager` with `<CUSTOM_NAME>`
5. Install the DaemonSet in your Kubernetes cluster using `kubeadm create -f kathara-daemonset.yml`.