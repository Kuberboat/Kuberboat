# Kuberboat

Kuberboat is a system which deploys and manages containerized applications. This is the course project of SJTU SE3356, 2022. 

## How to build

First, you should have Golang 1.18 installed. On MacOS, just run

```bash
brew install go@1.18
```

To generate the proto, you should install protobuf compiler and its Golang plugin. On MacOS, run

```bash
brew install protobuf@3.19
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

and set your `PATH` as 

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Now you are ready for building Kuberboat. Simply run

```bash
make
``` 

and you will see the executable under `out/bin`.

## How to run

### Prerequisite

You should install [Prometheus](https://prometheus.io/download/) on the master node (where API server is located). Please ensure that `prometheus` is placed under your system binary directory (`/usr/local/bin` on Linux). Simply check this by running

```bash
prometheus --version
```

Also, you should have a configuration file named `kubectl_condig.yaml` under your `$HOME/.kube/` directory. The example file is `test/examples/config.yaml`.

### Start Standalone

Just run the script below to start a standalone cluster:

```bash
scripts/kuberboat/start_standalone.sh
```

This will start the API Server, Kubelet, as well as all third-party tools required on one host machine. It is recommended that you run the script under root.

If you want to stop the standalone cluster, just run the script:

```bash
scripts/kuberboat/stop_standalone.sh
```


If you have [tmux](https://github.com/tmux/tmux) installed, start a standalone cluster by running the following command under the root directory of the project:

```bash
make start
```

This will start a tmux session with five windows:

1. shell
2. log of kubelet
3. log of apiserver
4. log of prometheus
5. log of etcd container

and register your local node to the apiserver.

Here are the commands for stop and restart:

```bash
make stop
make restart # equal to stop and start
```

### Start API Server

To start the API server, just run:

```bash
out/bin/apiserver
```

### Start Kubelet

To start Kubelet, just run:

```bash
out/bin/kubelet
```

We recommend that you run Kubelet as a superuser (i.e. root or `sudo`). Otherwise, you might find some errors after start.

## How to use

You could use `kubectl` command to interact with Kuberboat. We support following commands:

```bash
kubectl apply       # create component(s)
kubectl delete      # delete component(s)
kubectl describe    # show the details of component(s)
kubectl logs        # show the output of a job
```

You may use the help flag to get the usage. For example, enter 

```bash
kubectl apply -h
```

and you'll get the help information of `apply` command.
