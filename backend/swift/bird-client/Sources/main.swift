import AWSLambdaRuntime

let lambdaHandler = MyLambdaHandler()
let runtime = LambdaRuntime(lambdaHandler: lambdaHandler)
try await runtime.run()
