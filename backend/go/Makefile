SAM_BUILD_DIR ?= .aws-sam/build
BIN_DIR ?= bin

.PHONY: build-JobScraperFunction
build-JobScraperFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda

.PHONY: zip-lambda
zip-lambda: $(BIN_DIR)/lambda
	zip -j $(BIN_DIR)/lambda.zip $(BIN_DIR)/lambda

.PHONY: zip-lambda-bootstrap
zip-lambda-bootstrap: $(BIN_DIR)/lambda
	cp $(BIN_DIR)/lambda $(BIN_DIR)/bootstrap && zip -j $(BIN_DIR)/lambda.zip $(BIN_DIR)/bootstrap

$(BIN_DIR)/lambda:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BIN_DIR)/lambda ./cmd/lambda

.PHONY: clean
clean:
	rm -f $(BIN_DIR)/lambda $(BIN_DIR)/bootstrap $(BIN_DIR)/lambda.zip
	rm -rf $(SAM_BUILD_DIR)
