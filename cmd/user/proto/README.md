# proto
Package proto contains protocol buffer code to populate

## Generating client and server code
To generate the gRPC client and server interfaces from `*.proto` service definition.
Use the protocol buffer compiler protoc with a special gRPC Go plugin. For more info [read](https://grpc.io/docs/quickstart/go.html)

From this directory run:
```bash
$ make build
```
Running this command generates the following files in this directory:

* `*.pb.go`

This contains:

All the protocol buffer code to populate, serialize, and retrieve our request and response message types
An interface type (or stub) for clients to call with the methods defined in the services.
An interface type for servers to implement, also with the methods defined in the services.

* * *
Package proto contains protocol buffer code to populate
