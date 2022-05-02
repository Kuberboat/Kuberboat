# Kuberboat

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

### Prometheus

You should install [Prometheus](https://prometheus.io/download/) on the master node (where API server is located). Please ensure that `prometheus` is placed under your system binary directory (`/usr/local/bin` on Linux). Simply check this by running

```bash
prometheus --version
```

Then, run 

```bash
./scripts/prometheus/start_prom.sh
```

This script will start Prometheus based on the requirement of the project.

### API Server

To start the API server, just run

```bash
./out/bin/apiserver
```

### Kubelet

To start Kubelet, just run

```bash
./out/bin/kubelet
```

We recommend that you run Kubelet as a superuser (i.e. root or `sudo`). Otherwise, you might find some errors after start.


### Kubectl
You may use the help flag to get the usage.
```bash
kubectl apply -h
kubectl get -h
kubectl describe -h
kubectl delete -h
```

## All in one

You should still have Prometheus and ETCD installed. Under the root directory of the project, just run 
```bash
./scripts/kuberboat/start_standalone.sh
```
You'll get a standalone Kuberboat cluster including one API Server and one Kubelet on your localhost machine. Logs could be found in `./out/log`.
