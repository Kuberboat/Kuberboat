# Master Node Setup Guide

## Components

- [etcd](#etcd)
- [flannel](#flannel)

## etcd

The script below deletes current etcd data (if any) and runs etcd (v3.5.2) in a docker container. It is vital to **enable v2 API**, because Flannel does not support v3.

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

## flannel

Flannel daemons obtain information about the entire network from etcd. Here we will assign a subnet with IP range `172.17.0.0/16`, and each node will be given 256 IP addresses. We use vxlan to route packets, because it's quicker than UDP. Change `<etcd-address>` accordingly.

```bash
ETCDCTL_API=2 etcdctl --endpoints=<etcd-address> set /coreos.com/network/config '{"Network": "172.17.0.0/16", "SubnetLen": 24, "SubnetMin": "172.17.0.0","SubnetMax": "172.17.255.0", "Backend": {"Type": "vxlan"}}'
```

