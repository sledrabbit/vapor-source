SAM_BUILD_DIR ?= .aws-sam/build
BIN_DIR ?= bin
LAMBDA ?= scraper
LAMBDA_SRC := ./cmd/$(LAMBDA)
LAMBDA_OUT_DIR := $(BIN_DIR)/$(LAMBDA)
LAMBDA_BOOTSTRAP := $(LAMBDA_OUT_DIR)/bootstrap
LAMBDA_ZIP := $(LAMBDA_OUT_DIR)/lambda.zip

.PHONY: build-JobScraperFunction build-JobSnapshotFunction
build-JobScraperFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/scraper

build-JobSnapshotFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/snapshot

.PHONY: zip-lambda zip-scraper zip-snapshot
zip-lambda: $(LAMBDA_ZIP)

zip-scraper:
	$(MAKE) zip-lambda LAMBDA=scraper

zip-snapshot:
	$(MAKE) zip-lambda LAMBDA=snapshot

$(LAMBDA_ZIP): $(LAMBDA_BOOTSTRAP)
	zip -j $(LAMBDA_ZIP) $(LAMBDA_BOOTSTRAP)

$(LAMBDA_BOOTSTRAP):
	@mkdir -p $(LAMBDA_OUT_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(LAMBDA_BOOTSTRAP) $(LAMBDA_SRC)

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)/scraper $(BIN_DIR)/snapshot
	rm -rf $(SAM_BUILD_DIR)
