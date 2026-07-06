# grpc kern integration example

This example runs an HTTP kern app and a gRPC server in one process.

## Buf scaffold included

Proto contracts are under `proto/` and can be validated/generated with Buf.

### Install buf (once)

```bash
brew install bufbuild/buf/buf
```

### Validate proto contracts

```bash
buf lint
buf build
```

### Generate Go stubs

```bash
rm -rf pb
buf generate
```

This writes generated files to `pb/` using `source_relative` paths.

### Run the app

```bash
go run .
```

HTTP endpoint:

- `GET http://localhost:8080/health`

gRPC endpoint:

- `localhost:9090`
