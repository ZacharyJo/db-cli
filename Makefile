BINARY     := db-cli
MODULE     := github.com/ZacharyJo/db-cli
LDFLAGS    := -ldflags="-s -w"
BUILD_DIR  := bin

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build build-all build-target test clean fmt vet

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

build-target:
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

build-all:
	@mkdir -p $(BUILD_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) \
			-o $(BUILD_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT) . ; \
	)
	@echo "Built binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

test:
	go test ./... -v

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)/
