// swift-tools-version: 6.2
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
  name: "bird-client",
  platforms: [
    .macOS(.v15)
  ],
  dependencies: [
    .package(url: "https://github.com/awslabs/swift-aws-lambda-runtime.git", from: "2.4.0"),
    .package(url: "https://github.com/swift-server/swift-aws-lambda-events.git", from: "1.2.1"),
    .package(url: "https://github.com/scinfu/SwiftSoup.git", from: "2.11.1"),
    .package(url: "https://github.com/awslabs/aws-sdk-swift.git", from: "1.6.3"),
  ],
  targets: [
    // Targets are the basic building blocks of a package, defining a module or a test suite.
    // Targets can depend on other targets in this package and products from dependencies.
    .executableTarget(
      name: "bird-client",
      dependencies: [
        .product(name: "AWSSDKIdentity", package: "aws-sdk-swift"),
        .product(name: "AWSDynamoDB", package: "aws-sdk-swift"),
        .product(name: "AWSLambdaRuntime", package: "swift-aws-lambda-runtime"),
        .product(name: "AWSLambdaEvents", package: "swift-aws-lambda-events"),
        .product(name: "SwiftSoup", package: "SwiftSoup"),
      ]
    )
  ]
)
