# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gpop android ios gpop-cross evm all test clean
.PHONY: gpop-linux gpop-linux-386 gpop-linux-amd64 gpop-linux-mips64 gpop-linux-mips64le
.PHONY: gpop-linux-arm gpop-linux-arm-5 gpop-linux-arm-6 gpop-linux-arm-7 gpop-linux-arm64
.PHONY: gpop-darwin gpop-darwin-386 gpop-darwin-amd64
.PHONY: gpop-windows gpop-windows-386 gpop-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

gpop:
	$(GORUN) build/ci.go install ./cmd/gpop
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gpop\" to launch gpop."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gpop.aar\" to use the library."
	@echo "Import \"$(GOBIN)/gpop-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gpop.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/kevinburke/go-bindata/go-bindata@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gpop-cross: gpop-linux gpop-darwin gpop-windows gpop-android gpop-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gpop-*

gpop-linux: gpop-linux-386 gpop-linux-amd64 gpop-linux-arm gpop-linux-mips64 gpop-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-*

gpop-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gpop
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep 386

gpop-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gpop
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep amd64

gpop-linux-arm: gpop-linux-arm-5 gpop-linux-arm-6 gpop-linux-arm-7 gpop-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep arm

gpop-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gpop
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep arm-5

gpop-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gpop
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep arm-6

gpop-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gpop
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep arm-7

gpop-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gpop
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep arm64

gpop-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gpop
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep mips

gpop-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gpop
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep mipsle

gpop-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gpop
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep mips64

gpop-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gpop
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gpop-linux-* | grep mips64le

gpop-darwin: gpop-darwin-386 gpop-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gpop-darwin-*

gpop-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gpop
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-darwin-* | grep 386

gpop-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gpop
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-darwin-* | grep amd64

gpop-windows: gpop-windows-386 gpop-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gpop-windows-*

gpop-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gpop
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-windows-* | grep 386

gpop-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gpop
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gpop-windows-* | grep amd64
