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

1. Run `sudo ip link del flannel.1` on worker.
2. Run `sudo systemctl restart flannel` on worker. You may ignore the error.
3. Reconfigure docker as below. If the following commands don't work, please [follow the steps above](#configure-docker) one by one.
    ```bash
    sudo ./mk-docker-opts.sh` 
    sudo systemctl restart docker
    ```

4. You could delete the previous iptables rules using `sudo iptables -D ...` as you like.

### Troubleshooting

- All the above checking passes, but nodes still cannot communicate.

  Make sure the firewall/security group is configured properly. Also make sure nodes can ping each other.

### Learn Docker with Flannel by Example

#### 1. Before Everything

Make sure that **you have set up 2 workers as above on different hosts with the same flannel master**. Let them be `hostA` and `hostB`. Run a test docker on **each of them**. You may run:

```bash
docker run -d --name nginx --expose 80 nginx
```

#### 2. Take a Look at `iptables`

Run the following command on `hostA`:
```bash
sudo iptables -vL --line-number
```

You'll see:
```bash
Chain FORWARD
num  pkts    bytes   target                      prot  opt  in       out       source           destination     
1      0     0     DOCKER-USER                   all   --   any      any       anywhere         anywhere            
2      0     0     DOCKER-ISOLATION-STAGE-1      all   --   any      any       anywhere         anywhere 
3      0     0     ACCEPT                        all   --   any      docker0   anywhere         anywhere             ctstate RELATED,ESTABLISHED           
4      0     0     DOCKER                        all   --   any      docker0   anywhere         anywhere
5      0     0     ACCEPT                        all   --   docker0  !docker0  anywhere         anywhere          
6      0     0     ACCEPT                        all   --   docker0  docker0   anywhere         anywhere
7      0     0     ACCEPT                        all   --   any      any       10.17.0.0/16     anywhere             /* flanneld forward */
8      0     0     ACCEPT                        all   --   any      any       anywhere         10.17.0.0/16         /* flanneld forward */

......

Chain DOCKER (1 references)
num  pkts    bytes   target    prot  opt  in    out     source               destination

Chain DOCKER-ISOLATION-STAGE-1 (1 references)
num  pkts    bytes   target                      prot  opt  in        out       source               destination     
1      0     0       DOCKER-ISOLATION-STAGE-2    all   --   docker0   !docker0  anywhere             anywhere            
2      0     0       RETURN                      all   --   any       any       anywhere             anywhere            

Chain DOCKER-ISOLATION-STAGE-2 (1 references)
num  pkts    bytes   target    prot  opt  in     out      source             destination     
1      0     0       DROP       all  --   any    docker0  anywhere           anywhere            
2      0     0       RETURN     all  --   any    any      anywhere           anywhere    

Chain DOCKER-USER (1 references)
num  pkts    bytes   target    prot  opt  in     out      source             destination        
1    489     44304   RETURN     all  --   any    any      anywhere           anywhere
```

where `10.17.0.0/16` is the network range you set in master.

#### 3. Find out the IP Address Assigned by Flannel

Run
```bash
docker inspect nginx | grep Gateway
```
on `hostA`. You'll see:
```bash
"Gateway": "10.17.53.1",
```
This is the gateway IP that flannel assigns to `hostA`. Similarly, run
```bash
docker inspect nginx | grep IPAddress
```
on `hostA`. You'll see:
```bash
"IPAddress": "10.17.53.2",
```
This is the IP that flannel assigns to `nginx` docker. 

#### 4. `ping` Docker on Another Host

Run
```bash
ping 10.17.53.2
```
in the **`nginx` docker on `hostB`**. This IP is what flannel assigns to the `nginx` docker on `hostA`, as is shown above. You'll see:
```bash
PING 10.17.53.2 (10.17.53.2) 56(84) bytes of data.
64 bytes from 10.17.53.2: icmp_seq=1 ttl=62 time=2.36 ms
64 bytes from 10.17.53.2: icmp_seq=2 ttl=62 time=0.612 ms
64 bytes from 10.17.53.2: icmp_seq=3 ttl=62 time=0.573 ms
64 bytes from 10.17.53.2: icmp_seq=4 ttl=62 time=0.629 ms
^C
--- 10.17.53.2 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3041ms
rtt min/avg/max/mdev = 0.573/1.044/2.362/0.761 ms
```

Hooray! The 4 packets are all successfully sent to the docker on another host! We've verified that **flannel does create a network across different hosts, and the IPs it assigns to dockers work well!**

#### 4. Find out the Changes of `iptables`

Again, run the following command on `hostA`:
```bash
sudo iptables -vL --line-number
```

This time, you'll see:
```bash
Chain FORWARD
num  pkts    bytes   target                    prot  opt  in       out       source           destination     
1      8     672   DOCKER-USER                 all   --   any      any       anywhere         anywhere
2      8     672   DOCKER-ISOLATION-STAGE-1    all   --   any      any       anywhere         anywhere 
3      3     252   ACCEPT                      all   --   any      docker0   anywhere         anywhere             ctstate RELATED,ESTABLISHED           
4      1     84    DOCKER                      all   --   any      docker0   anywhere         anywhere
5      4     336   ACCEPT                      all   --   docker0  !docker0  anywhere         anywhere          
6      0     0     ACCEPT                      all   --   docker0  docker0   anywhere         anywhere
7      1     84    ACCEPT                      all   --   any      any       10.17.0.0/16     anywhere             /* flanneld forward */
8      0     0     ACCEPT                      all   --   any      any       anywhere         10.17.0.0/16         /* flanneld forward */

......

Chain DOCKER (1 references)
num  pkts    bytes   target     prot  opt  in     out      source             destination

Chain DOCKER-ISOLATION-STAGE-1 (1 references)
num  pkts    bytes   target                      prot  opt  in        out       source               destination     
1      4     336     DOCKER-ISOLATION-STAGE-2    all   --   docker0   !docker0  anywhere             anywhere            
2      8     672     RETURN                      all   --   any       any       anywhere             anywhere            

Chain DOCKER-ISOLATION-STAGE-2 (1 references)
num  pkts    bytes   target     prot  opt  in     out      source             destination     
1      0     0       DROP       all   --   any    docker0  anywhere           anywhere            
2      4     336     RETURN     all   --   any    any      anywhere           anywhere    

Chain DOCKER-USER (1 references)
num  pkts    bytes   target     prot  opt  in     out      source             destination
1    497     44976   RETURN     all   --   any    any      anywhere           anywhere
```

It could be indicated that in the host of `hostA`:
- The packets sent from `hostB` goes through:
  ```bash
  FORWARD 1
  DOCKER-USER 1
  FORWARD 2
  DOCKER-ISOLATION-STAGE-1 2

  # The first packet bypasses rule 3 and goes this way, since no connection 
  # has been established yet and `ctstate RELATED,ESTABLISHED` is not satisfied.
  1. FORWARD 4
     FORWARD 7 (ACCEPT)

  # The following packets go this way, since a connection has been established before.
  2. FORWARD 3 (ACCEPT)
  ```

- The response packets of `hostA` goes through:
  ```bash
  FORWARD 1
  DOCKER-USER 1
  FORWARD 2
  DOCKER-ISOLATION-STAGE-1 1    # Note that the response packets are from `docker0`.
  DOCKER-ISOLATION-STAGE-2 2
  DOCKER-ISOLATION-STAGE-1 2
  FORWARD 5 (ACCEPT)
  ```

#### 5. Learn More

[This URL](https://www.devopsschool.com/tutorial/kubernetes/kubernetes-cni-flannel-overlay-networking.html) might provide more information about how flannel works together with docker. If you prefer Chinese, [click here](https://www.zhihu.com/collection/794635077).
