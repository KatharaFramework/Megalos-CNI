# Megalos CNI

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

1. Pack the CNI in a binary file using `pyinstaller -F megalos.py`.
2. Move the generated CNI binary file in `bgp-manager/cni-bin`.
3. Build the Docker Image using `docker build -t <CUSTOM_NAME> .`
4. Push the Docker Image on your Docker Hub Repository using `docker push <CUSTOM_NAME>`.
5. Change the `kathara-daemonset.yml` file and replace `kathara/megalos-bgp-manager` with `<CUSTOM_NAME>`
6. Install the DaemonSet in your Kubernetes cluster using `kubeadm create -f kathara-daemonset.yml`.