# Master Node Setup Guide

## Components

- [Master Node Setup Guide](#master-node-setup-guide)
  - [Components](#components)
  - [Etcd](#etcd)
  - [Flannel](#flannel)
  - [Kuberboat API Server](#kuberboat-api-server)

## Etcd

The script below deletes current etcd data (if any) and runs etcd (v3.5.2) in a docker container. It is vital to **enable v2 API**, because flannel does not support v3.

```bash
#!/bin/bash
rm -rf /tmp/etcd-data.tmp && mkdir -p /tmp/etcd-data.tmp && \
docker run -d \
        --name etcd \
        -p 2379:2379 \
        --restart always \
        --mount type=bind,source=/tmp/etcd-data.tmp,destination=/etcd-data \
        quay.io/coreos/etcd:v3.5.2 \
        /usr/local/bin/etcd \
        --name s1 \
        --data-dir /etcd-data \
        --listen-client-urls http://0.0.0.0:2379 \
        --advertise-client-urls http://0.0.0.0:2379 \
        --listen-peer-urls http://0.0.0.0:2380 \
        --initial-advertise-peer-urls http://0.0.0.0:2380 \
        --initial-cluster s1=http://0.0.0.0:2380 \
        --initial-cluster-token tkn \
        --initial-cluster-state new \
        --enable-v2
```

To check if etcd is running properly, just check the log.

```bash
docker logs etcd
```

## Flannel

Flannel daemons obtain information about the entire network from etcd. Here we will assign a subnet with IP range `10.17.0.0/16`, and each node will be given 256 IP addresses. We use vxlan to route packets, because it's quicker than UDP. Change `<etcd-address>` accordingly.

```bash
ETCDCTL_API=2 etcdctl --endpoints=<etcd-address> set /coreos.com/network/config '{"Network": "10.17.0.0/16", "SubnetLen": 24, "SubnetMin": "10.17.0.0","SubnetMax": "10.17.255.0", "Backend": {"Type": "vxlan"}}'
```

## Kuberboat API Server

The following environment variables must be specified:

- `KUBE_SERVER_IP`: Tells the server about the node public IP. It is needed by the server to inform workers of its access point.
