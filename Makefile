# keenetic-smb-ui — build helpers
#
# Cross-compiles a static Go binary for every Entware architecture Keenetic
# routers use, then packs .ipk packages for opkg.

APP      := keenetic-smb-ui
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOOS     ?= linux
OUT      := dist
PKGROOT  := build/ipk

TARGETS := \
	linux/arm64 \
	linux/arm \
	linux/mipsle \
	linux/mips \
	linux/mips64le \
	linux/386 \
	linux/amd64

ARCH_IPK := \
	arm64:aarch64-3.10 \
	arm:armv7sf-k3.2 \
	mipsle:mipselsf-k3.4 \
	mips:mipssf-k3.4 \
	mips64le:mips64elsf-k3.4 \
	386:i386sf-k3.2 \
	amd64:x64-64

.PHONY: all build tidy test fmt vet clean ipk

all: build

tidy:
	go mod tidy

fmt:
	gofmt -l -w cmd internal

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p $(OUT)
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build -trimpath -ldflags "-s -w -X github.com/tommilevs/keenetic-smb-ui/internal/web.Version=$(VERSION)" \
			-o $(OUT)/$(APP)_$${os}_$${arch} ./cmd/keenetic-smb-ui || exit 1; \
	done
	@ls -lh $(OUT)

ipk: build
	@for pair in $(ARCH_IPK); do \
		arch=$${pair%%:*}; entware=$${pair#*:}; \
		bin=$(OUT)/$(APP)_linux_$$arch; \
		[ -f $$bin ] || continue; \
		root=$(PKGROOT)/$$entware; \
		rm -rf $$root; \
		mkdir -p $$root/opt/bin $$root/opt/etc/keenetic-smb-ui $$root/opt/etc/init.d $$root/CONTROL; \
		cp $$bin $$root/opt/bin/$(APP); \
		cp packaging/init.d/S99keenetic-smb-ui $$root/opt/etc/init.d/S99keenetic-smb-ui; \
		cp packaging/config.default.json $$root/opt/etc/keenetic-smb-ui/config.json; \
		sed -e "s/__VERSION__/$(VERSION)/g" -e "s/__ARCH__/$$entware/g" packaging/control > $$root/CONTROL/control; \
		printf "Package: $(APP)\nVersion: $(VERSION)\nArchitecture: $$entware\n" > $$root/CONTROL/control; \
		echo "/opt/bin/$(APP)" > $$root/CONTROL/conffiles 2>/dev/null || true; \
		( cd $$root && tar --owner=0 --group=0 -czf ../../../$(OUT)/$(APP)_$(VERSION)_$$entware.ipk ./CONTROL ./opt ); \
	done
	@ls -lh $(OUT)/*.ipk

clean:
	rm -rf $(OUT) build
