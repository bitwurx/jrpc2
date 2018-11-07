# Golang JSON-RPC 2.0 over HTTP Service Library

This library implements JSON-RPC 2.0 over HTTP service loosely following specification:

 - [JSON-RPC 2.0 Specification](http://www.jsonrpc.org/specification). 
 - [JSON-RPC 2.0 over HTTP Specification Draft](https://www.simple-is-better.org/json-rpc/transport_http.html)

### About

**s3rj1k/jrpc2** was originally based on Jared Patricks **[bitwurx/jrpc2](https://github.com/bitwurx/jrpc2)**,
but has been completely rewritten.

### Known limitations:
 - no support for batch requests

### Installation
```sh
go get github.com/s3rj1k/jrpc2
```

### Examples
 - https://gist.github.com/s3rj1k/b45b47b0e80f215e459974507a528d8e
 - see tests for other usage examples.

### Running Tests

This library contains a set of API tests to verify 
specification compliance, at least it tries to be compliant.

```sh
go test ./... -v
```
