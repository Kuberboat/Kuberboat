BUILD_DIR = ./out/bin
CMD_SOURCE_DIRS = cmd
SOURCE_DIRS = cmd pkg
SOURCE_PACKAGES = ./cmd/... ./pkg/...
BOAT_SRC = ./cmd/boat/main.go
BOAT_OBJ = boat

$(shell mkdir -p $(BUILD_DIR))

export GO111MODULE := on

boat: $(BOAT_SRC)
	@go build -o $(BUILD_DIR)/$(BOAT_OBJ) $(BOAT_SRC)

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
