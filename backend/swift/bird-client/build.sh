#!/bin/bash
set -e

# Build in Amazon Linux 2 container
swift package archive --allow-network-connections docker

# Copy over prompt.txt 
zip -j .build/plugins/AWSLambdaPackager/outputs/AWSLambdaPackager/bird-client/bird-client.zip prompt.txt
