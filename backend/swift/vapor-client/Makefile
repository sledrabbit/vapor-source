.PHONY: build-VaporClientFunction clean

EXECUTABLE_NAME=vapor-client
BUILD_CONFIG=release
TARGET_ARCH=aarch64

ZIP_FILE=$(ARTIFACTS_DIR)Package.zip

SWIFT_BUILD_DIR=.build/$(TARGET_ARCH)-unknown-linux-gnu/$(BUILD_CONFIG)

SWIFT_VERSION=6.1
SWIFT_PLATFORM=amazonlinux2-aarch64
SWIFT_BRANCH=swift-$(SWIFT_VERSION)-RELEASE
SWIFT_URL=https://download.swift.org/swift-6.1-release/$(SWIFT_PLATFORM)/$(SWIFT_BRANCH)/$(SWIFT_BRANCH)-$(SWIFT_PLATFORM).tar.gz
SWIFT_DIR_NAME=$(SWIFT_BRANCH)-$(SWIFT_PLATFORM)
SWIFT_INSTALL_DIR=/tmp/swift-$(SWIFT_VERSION)

build-VaporClientFunction:
	mkdir -p $(SWIFT_INSTALL_DIR)
	curl -fsSL $(SWIFT_URL) -o $(SWIFT_INSTALL_DIR)/swift.tar.gz
	tar -xzf $(SWIFT_INSTALL_DIR)/swift.tar.gz -C $(SWIFT_INSTALL_DIR)
	export PATH=$(SWIFT_INSTALL_DIR)/$(SWIFT_DIR_NAME)/usr/bin:"$$PATH"; \
	swift build --configuration $(BUILD_CONFIG) --arch $(TARGET_ARCH); \
	mkdir -p "$(ARTIFACTS_DIR)"; \
	cp "$(SWIFT_BUILD_DIR)/$(EXECUTABLE_NAME)" "$(ARTIFACTS_DIR)/bootstrap"; \
	cp "prompt.txt" "$(ARTIFACTS_DIR)/prompt.txt"; \
	cp -L $(SWIFT_INSTALL_DIR)/$(SWIFT_DIR_NAME)/usr/lib/swift/linux/*.so "$(ARTIFACTS_DIR)/"

package: build-VaporClientFunction
	cd "$(ARTIFACTS_DIR)" && zip -r ../$(notdir $(ZIP_FILE)) .
	@echo "Lambda package created at: $(ZIP_FILE)"

clean:
	swift package clean || true
	rm -rf $(SWIFT_INSTALL_DIR)
	rm -rf .aws-sam
	rm -f $(ZIP_FILE)