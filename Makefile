BUILD_DIR = ./out/bin
CMD_SOURCE_DIRS = cmd
SOURCE_DIRS = cmd pkg
SOURCE_PACKAGES = ./cmd/... ./pkg/...
APISERVER_SRC = ./cmd/apiserver/apiserver.go
APISERVER_OBJ = apiserver
PROTO_SCRIPT = scripts/proto_gen.sh

$(shell mkdir -p $(BUILD_DIR))

export GO111MODULE := on
export GOPROXY := https://mirrors.aliyun.com/goproxy/,direct

all: proto $(APISERVER_SRC)
	@go build -o $(BUILD_DIR)/$(APISERVER_OBJ) $(APISERVER_SRC)

.PHONY: proto
proto:
	./$(PROTO_SCRIPT)

.PHONY: fmt
fmt:
	@gofmt -s -w $(SOURCE_DIRS)

.PHONY: imports
imports:
	@goimports -w $(SOURCE_DIRS)

.PHONY: vet
vet:
	@go vet $(SOURCE_PACKAGES)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
