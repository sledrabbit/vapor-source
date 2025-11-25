SAM_BUILD_DIR ?= .aws-sam/build
BIN_DIR ?= bin

.PHONY: build-JobScraperFunction
build-JobScraperFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda

.PHONY: zip-lambda
zip-lambda: $(BIN_DIR)/bootstrap
	zip -j $(BIN_DIR)/lambda.zip $(BIN_DIR)/bootstrap

$(BIN_DIR)/bootstrap:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BIN_DIR)/bootstrap ./cmd/lambda

.PHONY: clean
clean:
	rm -f $(BIN_DIR)/lambda $(BIN_DIR)/bootstrap $(BIN_DIR)/lambda.zip
	rm -rf $(SAM_BUILD_DIR)
