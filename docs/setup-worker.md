# Worker Node Setup Guide

## Components

- [Docker](#docker)
- [Flannel](#flannel)

## Docker

Check [official website](https://docs.docker.com/engine/install/ubuntu/) to setup Docker.

## Flannel

We use flannel v0.17.0 and vxlan to forward packets.

### Prerequisite

Make sure [the flannel network config is set in etcd](setup-master.md#flannel).

### Download Flannel

```bash
wget https://github.com/flannel-io/flannel/releases/download/v0.17.0/flannel-v0.17.0-linux-amd64.tar.gz
tar -xvf flannel-v0.17.0-linux-amd64.tar.gz
sudo mv flanneld /usr/bin
```

### Run Flanneld as a service

1. Create a flannel config file under `/etc/flannel/flanneld.env` with the following content. Change `<etcd-endpoint>` to your own etcd address.

    ```bash
    FLANNEL_OPTS="--etcd-endpoints=<etcd-endpoint>
    ```

2. Create a flannel service config file under `/etc/systemd/system/flannel.service` with the following content.

    ```bash
    [Unit]
    Description=Flannel daemon
    After=network.target network-online.target
    Before=docker.service

    [Service]
    Type=notify
    User=root
    EnvironmentFile=/etc/flannel/flanneld.env
    ExecStart=/usr/bin/flanneld $FLANNEL_OPTS
    Restart=on-failure

    [Install]
    WantedBy=multi-user.target
    ```

3. Then start flannel service.

    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable flannel
    ```

4. To verify that flannel is working properly, use the following commands.

    ```bash
    # Make sure flannel daemon is running.
    sudo systemctl status -l flannel
    # Make sure flannel device is created.
    ifconfig
    # Make sure routing table is configured.
    route -n | grep flannel
    # Make sure flannel has generated subnet config file.
    cat /run/flannel/subnet.env
    ```

### Configure Docker

#### 1. Inform Docker to Use the Subnet 

To generate Docker config file, run the script `mk-docker-opts.sh` shipped with flannel.

```bash
sudo ./mk-docker-opts.sh
```

The file shoud be generated in `/run/docker_opts.env`. 

#### 2. Modify Docker service
    
Docker service is configured under `/lib/systemd/system/docker.service`. 
- Change `ExecStart=/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock` to `ExecStart=/usr/bin/dockerd $DOCKER_OPTS -H fd:// --containerd=/run/containerd/containerd.sock`
- Add a line `EnvironmentFile=/run/docker_opts.env` just above that line.

#### 3. Restart Docker Daemon

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```
#### 4. Verify the Result

To verify that Docker is now using the subnet assigned by flannel, use the following commands.

```bash
# Make sure the default bridge network uses flannel-assigned subnet.
docker network inspect bridge
# Make sure flannel.1 and docker0 are in the same subnet.
ifconfig
```

At this point the containers across nodes with flannel set up should be able to communicate with each other.

### Reconfigure Flannel with Master

1. Run `sudo systemctl restart flannel` on worker. You may ignore the error.
2. [Reconfigure docker](#configure-docker) as above. You may only need to run `sudo ./mk-docker-opts.sh` in step 1 and `sudo systemctl restart docker` in step 3. If this doesn't work, please follow the steps above one by one.

### Troubleshooting

- All the above checking passes, but nodes still cannot communicate.

  Make sure the firewall/security group is configured properly. Also make sure nodes can ping each other.
