APP_NAME := webhookd
ZOOM_APP := zoomwebhookd

GO      ?= go
BIN_DIR := bin

.PHONY: all build build-zoom test lint tidy clean run-zoom build-stress run-stress

all: build build-zoom

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN_DIR)/$(APP_NAME) ./cmd/webhookd

build-zoom:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN_DIR)/$(ZOOM_APP) ./cmd/zoomwebhookd

test:
	$(GO) test ./... -v -count=1

lint:
	@echo "tip: use golangci-lint locally; CI runs static checks"

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)

build-stress:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN_DIR)/stresszoom ./cmd/stresszoom

run-stress: build-stress
	ZOOM_WEBHOOK_SECRET_TOKEN?=supersecret
	ZOOM_WEBHOOK_SECRET_TOKEN=$${ZOOM_WEBHOOK_SECRET_TOKEN} ./bin/stresszoom -url http://127.0.0.1:8080/webhook/zoom -rate 3000 -conns 800 -duration 60s -body 512 -workers 32
