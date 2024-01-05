# Golang AWS SAM Lambda example
This example project shows how to use AWS SAM with two Golang Lambdas.

## Quickstart
After installing AWS SAM, run the following command to build the containers and run the service locally:
```bash
sam build && sam local start-api
```

Remote debugging inside these containers is also possible. Please read through the command line arguments
of `sam local start-api` to find the options that work with your system.

## Files
* `cmd` holds the executables that can be built and put into AWS Lambdas.
* `cmd/one` and `cmd/two` are distinctly different Lambdas that handle different API URL paths.
* `util` is a helper package that exports typed & wrapped HTTP Lambda interfaces to implement. It's helpful to have the
  input and outputs as typed Go structures, rather than `[]byte`.
* `template.yml` is a template used by AWS SAM for local deployments. AWS SAM can also use this template to deploy to
  production, but that's not covered in this example project.

## Lambda one
This Lambda has shows how to handle environment variables and HTTP request metadata like the requester's source IP. It
also has an external API request example.

Its HTTP method is `GET`.

Example `curl`:
```bash
$ curl http://localhost:3000/one
{
    "customString": "This is a value from an environment variable for Lambda one.",
    "randomPokemon": "raikou",
    "sourceIP": "127.0.0.1",
    "time": "2022-01-13T17:42:22.839696271Z",
    "userAgent": "Custom User Agent String"
}
```

## Lambda two
This Lambda shows how to use custom path parameters and how data can be saved across a single Lambda _instance_'s
_invocations_.

Its HTTP method is `POST`.

Example `curl`:
```bash
$ curl --request POST http://localhost:3000/two/monkey
{
    "customPath": "monkey",
    "prevCustomPath": ""
}
```

## Debugging
Confirm `dlv` is installed:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Go remote debugging must be started shortly before or after AWS SAM gets a request. If a breakpoint is triggered, `dlv`
or the compatible IDE can control and inspect the execution.

Due to and AWS SAM bug and the lack of support for building Go Lambda in containers, newer operating systems such as
Ubuntu 21.04+ and Fedora 35+ will not be able to use AWS SAM's remote debugging feature. This has to do with the
`glibc` version. See [this GitHub issue](https://github.com/aws/aws-sam-cli/issues/2294).

`Error: We do not support building Go Lambda functions within a container. Try building without the container. Most Go functions will build successfully.`

## Troubleshooting
### General
#### The wrong HTTP method was used, but the Lambda still processed the request
AWS API Gateway request routing is pretty flexible, don't expect the request routing rules in `template.yml` or an
OpenAPI specification to be followed strictly by API Gateway.

#### Switching the Lambda to use API Gateway V2 makes some parts of the request empty
This is because AWS SAM is only giving the API Gateway V1 format. I _think_ `Type: "AWS::Serverless::Function"` needs to
be changed in `template.yml` to something that indicates it's a Lambda that is triggered by API Gateway V2. It might
also be the type of `ApiEvent`. I'm not sure if AWS SAM supports that use case yet.

### Lambda one
#### The User Agent returned is always `Custom User Agent String`
I think this is specific to AWS SAM. I believe the developers didn't include logic to extract that from the request.

#### The Pokemon returned is always `trubbish API`
This means the Lambda was unable to successfully get a Pokemon from an external API.

This could be because it failed to connect over the internet. But a more likely scenario is a known bug in AWS SAM. This
bug causes AWS SAM to always pass an expired `context.Context` to the Lambda handler. Since the context is always
expired before the request is sent, it will never send. This has been fixed in more recent versions of AWS SAM, so
upgrade your version if you are affected by it.
Relevant [GitHub issue](https://github.com/aws/aws-sam-cli/issues/2510#issuecomment-827497820).

### Lambda two
#### The value of `prevCustomPath` returned by the API is always an empty string or seemingly wrong
This is because AWS SAM spins up and spins down a single Lambda _instance_ for every request. In production AWS will
spin up Lambdas until the number of requests per second coming in dictate that _instances_ should be spun down. This is
an example of how data can be shared across _invocations_, but not _instances_.
