# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gups deps android ios gups-cross swarm evm all test clean
.PHONY: gups-linux gups-linux-386 gups-linux-amd64 gups-linux-mips64 gups-linux-mips64le
.PHONY: gups-linux-arm gups-linux-arm-5 gups-linux-arm-6 gups-linux-arm-7 gups-linux-arm64
.PHONY: gups-darwin gups-darwin-386 gups-darwin-amd64
.PHONY: gups-windows gups-windows-386 gups-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest
DEPS = $(shell pwd)/internal/jsre/deps

gups:
	build/env.sh go run build/ci.go install ./cmd/gups
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gups\" to launch gups."

deps:
	cd $(DEPS) &&	go-bindata -nometadata -pkg deps -o bindata.go bignumber.js web3.js
	cd $(DEPS) &&	gofmt -w -s bindata.go
	@echo "Done generate deps."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

# android:
#	build/env.sh go run build/ci.go aar --local
#	@echo "Done building."
#	@echo "Import \"$(GOBIN)/gups.aar\" to use the library."

# ios:
#	build/env.sh go run build/ci.go xcode --local
#	@echo "Done building."
#	@echo "Import \"$(GOBIN)/Getrue.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gups-cross: gups-linux gups-darwin gups-windows gups-android gups-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gups-*

gups-linux: gups-linux-386 gups-linux-amd64 gups-linux-arm gups-linux-mips64 gups-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-*

gups-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gups
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep 386

gups-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gups
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep amd64

gups-linux-arm: gups-linux-arm-5 gups-linux-arm-6 gups-linux-arm-7 gups-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep arm

gups-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gups
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep arm-5

gups-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gups
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep arm-6

gups-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gups
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep arm-7

gups-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gups
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep arm64

gups-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gups
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep mips

gups-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gups
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep mipsle

gups-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gups
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep mips64

gups-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gups
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gups-linux-* | grep mips64le

gups-darwin: gups-darwin-386 gups-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gups-darwin-*

gups-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gups
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gups-darwin-* | grep 386

gups-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gups
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gups-darwin-* | grep amd64

gups-windows: gups-windows-386 gups-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gups-windows-*

gups-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gups
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gups-windows-* | grep 386

gups-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gups
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gups-windows-* | grep amd64
