#!/bin/bash
set -e

# Build in Amazon Linux 2 container with docker
swift package archive --disable-sandbox --allow-network-connections docker

# using apple container
# swift package --disable-sandbox --allow-network-connections docker archive --container-cli container

# Copy over prompt.txt
# zip -j .build/plugins/AWSLambdaPackager/outputs/AWSLambdaPackager/bird-client/bird-client.zip prompt.txt
