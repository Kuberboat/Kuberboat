BUILD_DIR = ./out/bin
CMD_SOURCE_DIRS = cmd
SOURCE_DIRS = cmd pkg
SOURCE_PACKAGES = ./cmd/... ./pkg/...
APISERVER_SRC = ./cmd/apiserver/apiserver.go
APISERVER_OBJ = apiserver
PROTO_SCRIPT = scripts/proto_gen.sh
PROTO_GEN_DIR = ./pkg/proto
KUBELET_SRC = ./cmd/kubelet/kubelet.go
KUBELET_OBJ = kubelet
KUBECTL_SRC = ./cmd/kubectl/kubectl.go
KUBECTL_OBJ = kubectl
SCRIPTS_DIR = ./scripts/kuberboat
SHFMT_FLAG = shfmt
XARGS_FLAG = xargs

$(shell mkdir -p $(BUILD_DIR))

export GO111MODULE := on
export GOPROXY := https://mirrors.aliyun.com/goproxy/,direct

all: proto apiserver kubelet kubectl

apiserver: $(APISERVER_SRC)
	@go build -o $(BUILD_DIR)/$(APISERVER_OBJ) $(APISERVER_SRC)

kubelet: $(KUBELET_SRC)
	@go build -o $(BUILD_DIR)/$(KUBELET_OBJ) $(KUBELET_SRC)

kubectl: $(KUBECTL_SRC)
	@go build -o $(BUILD_DIR)/$(KUBECTL_OBJ) $(KUBECTL_SRC)

.PHONY: proto
proto:
	rm -rf $(PROTO_GEN_DIR)
	./$(PROTO_SCRIPT)

.PHONY: fmt
fmt:
	@gofmt -s -w $(SOURCE_DIRS)
	$(SHFMT_FLAG) -f . | $(XARGS_FLAG) $(SHFMT_FLAG) -w

.PHONY: imports
imports:
	@goimports -w $(SOURCE_DIRS)

.PHONY: vet
vet:
	@go vet $(SOURCE_PACKAGES)

.PHONY: clean
clean:
	rm -rf $(PROTO_GEN_DIR)
	rm -rf $(BUILD_DIR)
.PHONY: start
start:
	$(SCRIPTS_DIR)/kuberboat-dev.sh start
.PHONY: stop
stop:
	$(SCRIPTS_DIR)/kuberboat-dev.sh stop
.PHONY: restart
restart:
	$(SCRIPTS_DIR)/kuberboat-dev.sh restart