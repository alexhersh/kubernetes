<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/getting-started-guides/fedora/fedora-calico.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->
Running Kubernetes with [Calico Networking](http://projectcalico.org) on a [Digital Ocean](http://digitalocean.com) [Fedora Host](http://fedoraproject.org)
-----------------------------------------------------

## Prerequisites

You need 2 or more Fedora droplets on Digital Ocean with [Private Networking](https://www.digitalocean.com/community/tutorials/how-to-set-up-and-use-digitalocean-private-networking) enabled.

## Limitations

- Current fedora kubernetes release is still 0.20.0. You'll need to manually download and install kubernetes 1.0.0 (despite what these instructions say) until the package is updated.

## Overview

This guide will walk you through the process of getting a Kubernetes Fedora cluster running on Digital Ocean with networking powered by Calico networking. It will cover the installation and configuration of the following systemd processes on the following hosts:

Kubernetes Master:
- `kube-apiserver`
- `kube-controller-manager`
- `kube-scheduler`
- `etcd`
- `docker`
- `calico-node`

Kubernetes Node:
- `kubelet`
- `kube-proxy`
- `docker`
- `calico-node`

For this demo, we will be setting up one Master and one Node with the following information:

|  Hostname   |     IP      |
|-------------|-------------|
| kube-master |10.134.251.56|
| kube-node-1 |10.134.251.55|

This guide is scallable to multiple nodes provided you [configure interface-cbr0 with its own subnet on each Node](#create-the-new-virtual-interface) and [add an entry to /etc/hosts for each host](#setup-communication-between-hosts).

Ensure you substitute the IP Addresses and Hostnames used in this guide with ones in your own setup. 

### Setup Communication Between Hosts

Digital Ocean private networking configures a private network on eth1 for each host.  To simplify communication between the hosts, we will add an entry to /etc/hosts so that all hosts in the cluster can hostname-resolve one another to this interface.  **It is important that the hostname resolves to this interface instead of eth0, as all Kubernetes and Calico services will be running on it.**

```
echo "10.134.251.56 kube-master" >> /etc/hosts
echo "10.134.251.55 kube-node-1" >> /etc/hosts
```

>Make sure that communication works between kube-master and each kube-node by using a utility such as ping.

### Install Kubernetes on Master

* Run the following command on Master to install the latest Kubernetes (as well as docker):

```
yum -y install --enablerepo=updates-testing kubernetes
```

* Edit /etc/kubernetes/config 

```
# How the controller-manager, scheduler, and proxy find the apiserver
KUBE_MASTER="--master=http://kube-master:8080"
```

## Configure Master

* Both Calico and Kubernetes use etcd as their datastore. We will run etcd on Master and point all kubernetes and calico services to it.

```
yum -y install etcd
```

* Edit /etc/etcd/etcd.conf

```
ETCD_LISTEN_CLIENT_URLS="http://kube-master:4001"

ETCD_ADVERTISE_CLIENT_URLS="http://kube-master:4001"
```

* Edit /etc/kubernetes/apiserver

```
# The address on the local server to listen to.
KUBE_API_ADDRESS="--address=http://kube-master"

KUBE_ETCD_SERVERS="--etcd_servers=http://kube-master:4001"

# Remove ServiceAccount from this line to run without API Tokens
KUBE_ADMISSION_CONTROL="--admission_control=NamespaceLifecycle,NamespaceExists,LimitRanger,SecurityContextDeny,ResourceQuota"
```

* Start the appropriate services on master:

```
for SERVICES in etcd kube-apiserver kube-controller-manager kube-scheduler; do
    systemctl restart $SERVICES
    systemctl enable $SERVICES
    systemctl status $SERVICES
done
```

## Configure the Node

## Install Kubernetes on Nodes

* Run the following command on the Node to install the latest Kubernetes services (as well as docker):

```
yum -y install --enablerepo=updates-testing kubernetes
```

* Edit /etc/kubernetes/config

```
# How the controller-manager, scheduler, and proxy find the apiserver
KUBE_MASTER="--master=http://kube-master:8080"

# The following are variables which the kubelet will pass to the calico-networking plugin
# Network path to ETCD datastore
ETCD_AUTHORITY="kube-master:4001"

# Network path to the api-server
KUBE_API_ROOT="http://kube-master:8080/api/v1"
```

* Edit /etc/kubernetes/kubelet

```
# The address for the info server to serve on (set to 0.0.0.0 or "" for all interfaces)
KUBELET_ADDRESS="--address=0.0.0.0"

# You may leave this blank to use the actual hostname
# KUBELET_HOSTNAME="--hostname_override=127.0.0.1"

# location of the api-server
KUBELET_API_SERVER="--api_servers=http://kube-master:8080"

# Add your own!
KUBELET_ARGS="--network-plugin=calico"
```

Before starting the Kubernetes services, we will make some configuration changes to Docker.

### Create the new Virtual Interface

* Add a virtual interface by creating `/etc/sysconfig/network-scripts/ifcfg-cbr0`:

```
DEVICE=cbr0
TYPE=Bridge
IPADDR=192.168.1.1
NETMASK=255.255.255.0
ONBOOT=yes
BOOTPROTO=static
```

**Note for Multi-Node Clusters:** Each node should have an IP address on a different subnet. In this example, node-1 is using 192.168.1.1/24, so node-2 should be assigned another pool on the 192.168.x.0/24 subnet. For example: 192.168.2.0/24.

* Ensure that your system has bridge-utils installed. Then, restart the networking daemon to activate the new interface

```
systemctl restart network.service
```

### Start Docker on the new Virtual Interface

* Configure docker to run on the new vnic by editing `/etc/sysconfig/docker-network`:

```
DOCKER_NETWORK_OPTIONS="--bridge=cbr0 --iptables=false --ip-masq=false"
```

* Start docker

```
systemctl start docker
```

### Install & Start Calico

* Install calicoctl, the calico-kubernetes helper binary.

```
wget https://github.com/Metaswitch/calico-docker/releases/download/v0.5.1/calicoctl
chmod +x ./calicoctl
sudo mv ./calicoctl /usr/bin
```

* Create /etc/systemd/calico-node.service

```
[Unit]
Description=calicoctl node
Requires=docker.service
After=docker.service

[Service]
User=root
Environment="ETCD_AUTHORITY=kube-master:4001"
PermissionsStartOnly=true
ExecStartPre=/usr/bin/calicoctl checksystem --fix
ExecStart=/usr/bin/calicoctl node --ip=10.134.251.55 --detach=false --kubernetes

[Install]
WantedBy=multi-user.target
```

**Note: You must the IP address with your node's eth1 IP Address!**

* Start Calico

```
systemctl enable /etc/systemd/calico-node.service
systemctl start calico-node.service
```

## Start the Services on Node

* Start the appropriate services on the node (kube-node-1).

```
for SERVICES in kube-proxy kubelet; do 
    systemctl restart $SERVICES
    systemctl enable $SERVICES
    systemctl status $SERVICES 
done
```

The cluster should be running! Check that your nodes are reporting as such:

```
kubectl get nodes
NAME          LABELS                               STATUS
kube-node-1   kubernetes.io/hostname=kube-node-1   Ready

```

## Launching an Application with Calico - a Note on DNS

Most Kubernetes application deployments will require a DNS server hosted within the kubernetes cluster. In order to function properly, this DNS container will need to communicate with the kube-apiserver to gather a list of active kubernetes services. However, requests sent from the skydns container will not be returned by the kube-apiserver as the Digital Ocean networking fabric will drop response packets destined for any 192.168.0.0/16 address. To resolve this, you can have calicoctl add a masquerade rule to all outgoing traffic on the node:

```
ETCD_AUTHORITY=kube-master:4001 calicoctl pool add 192.168.0.0/16 --nat-outgoing
```


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/getting-started-guides/fedora/fedora-calico.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
