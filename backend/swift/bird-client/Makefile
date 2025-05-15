.PHONY: build-BirdClientFunction clean package

EXECUTABLE_NAME=bird-client
BUILD_CONFIG=release
TARGET_ARCH=aarch64

ZIP_FILE=bird-client-lambda.zip
SAM_BUILD_OUTPUT_DIR = .aws-sam/build/BirdClientFunction # Used by 'package' target

SWIFT_BUILD_DIR=.build/$(TARGET_ARCH)-unknown-linux-gnu/$(BUILD_CONFIG)

# Swift Configuration
SWIFT_VERSION := 6.1
SWIFT_PLATFORM := amazonlinux2-aarch64
SWIFT_INSTALL_ROOT := /opt/swift/swift-$(SWIFT_VERSION)-RELEASE-$(SWIFT_PLATFORM)
SWIFT_EXECUTABLE := $(SWIFT_INSTALL_ROOT)/usr/bin/swift
SWIFT_LIB_DIR := $(SWIFT_INSTALL_ROOT)/usr/lib/swift/linux
export PATH := $(SWIFT_INSTALL_ROOT)/usr/bin:$(PATH)

build-BirdClientFunction:
	@echo "Using Swift: $(SWIFT_EXECUTABLE)"
	@"$(SWIFT_EXECUTABLE)" --version || { echo "Error: '$(SWIFT_EXECUTABLE) --version' failed."; exit 1; }
	@"$(SWIFT_EXECUTABLE)" build --configuration $(BUILD_CONFIG) --arch $(TARGET_ARCH) || { echo "Error: Swift build failed."; exit 1; }

	@echo "Preparing artifacts in $(ARTIFACTS_DIR)..."
	mkdir -p "$(ARTIFACTS_DIR)"
	cp "$(SWIFT_BUILD_DIR)/$(EXECUTABLE_NAME)" "$(ARTIFACTS_DIR)/bootstrap"
	cp "prompt.txt" "$(ARTIFACTS_DIR)/prompt.txt"

	@echo "Copying Swift runtime libraries from $(SWIFT_LIB_DIR)..."
	cp -L $(SWIFT_LIB_DIR)/*.so "$(ARTIFACTS_DIR)/" 2>/dev/null || true
	if [ -d "$(SWIFT_LIB_DIR)/$(TARGET_ARCH)" ]; then \
	    echo "Copying arch-specific libraries from $(SWIFT_LIB_DIR)/$(TARGET_ARCH)..."; \
	    cp -L $(SWIFT_LIB_DIR)/$(TARGET_ARCH)/*.so "$(ARTIFACTS_DIR)/" 2>/dev/null || true; \
	fi

	@echo "Build and library packaging successful for $(EXECUTABLE_NAME)."

package:
	cd "$(SAM_BUILD_OUTPUT_DIR)" && zip -r "$(CURDIR)/$(ZIP_FILE)" .
	@echo "Lambda package created: $(CURDIR)/$(ZIP_FILE)"

clean:
	@echo "Cleaning project..."
	swift package clean || true
	rm -rf .build
	rm -rf .aws-sam
	rm -f $(ZIP_FILE)
