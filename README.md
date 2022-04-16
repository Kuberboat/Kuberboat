# Kuberboat

## How to build

First, you should have Golang 1.18 installed. On MacOS, just run

```shell
brew install go@1.18
```

To generate the proto, you should install protobuf compiler and its Golang plugin. On MacOS, run

```shell
brew install protobuf@3.19
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

and set your `PATH` as 

```shell
export PATH="$PATH:$(go env GOPATH)/bin"
```

Now you are ready for building Kuberboat. Simply run

```shell
make
``` 

and you will see the executable under `out/bin`.

### Kubectl
You may use the help flag to get the usage.
```shell
kubectl apply -h
kubectl get -h
kubectl describe -h
kubectl delete -h
```