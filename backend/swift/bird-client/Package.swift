// swift-tools-version: 6.2
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
  name: "bird-client",
  platforms: [
    .macOS(.v15)
  ],
  dependencies: [
    .package(
      url: "https://github.com/swift-server/swift-aws-lambda-runtime.git", from: "2.1.0"
    ),
    .package(url: "https://github.com/swift-server/swift-aws-lambda-events.git", from: "1.2.1"),
    .package(url: "https://github.com/tid-kijyun/Kanna.git", from: "6.0.1"),
  ],
  targets: [
    // Targets are the basic building blocks of a package, defining a module or a test suite.
    // Targets can depend on other targets in this package and products from dependencies.
    .executableTarget(
      name: "bird-client",
      dependencies: [
        .product(name: "AWSLambdaRuntime", package: "swift-aws-lambda-runtime"),
        .product(name: "AWSLambdaEvents", package: "swift-aws-lambda-events"),
        .product(name: "Kanna", package: "Kanna"),
      ]
    )
  ]
)
