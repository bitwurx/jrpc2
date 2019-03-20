# Golang JSON-RPC 2.0 HTTP Server

[![GoDoc](https://godoc.org/github.com/bitwurx/jrpc2?status.png)](https://godoc.org/github.com/bitwurx/jrpc2)

This library is an HTTP server implementation of the [JSON-RPC 2.0 Specification](http://www.jsonrpc.org/specification). The library is fully spec compliant with support for named and positional arguments and batch requests.

### Installation
```sh
go get github.com/bitwurx/jrpc2
```

### Quickstart

```golang
package main

import (
    "encoding/json"
    "errors"
    "os"

    "github.com/bitwurx/jrpc2"
)

// This struct is used for unmarshaling the method params
type AddParams struct {
    X *float64 `json:"x"`
    Y *float64 `json:"y"`
}

// Each params struct must implement the FromPositional method.
// This method will be passed an array of interfaces if positional parameters
// are passed in the rpc call
func (ap *AddParams) FromPositional(params []interface{}) error {
    if len(params) != 2 {
        return errors.New("exactly two integers are required")
    }

    x := params[0].(float64)
    y := params[1].(float64)
    ap.X = &x
    ap.Y = &y

    return nil
}

// Each method should match the prototype <fn(json.RawMessage) (inteface{}, *ErrorObject)>
func Add(params json.RawMessage) (interface{}, *jrpc2.ErrorObject) {
    p := new(AddParams)

    // ParseParams is a helper function that automatically invokes the FromPositional
    // method on the params instance if required
    if err := jrpc2.ParseParams(params, p); err != nil {
        return nil, err
    }

    if p.X == nil || p.Y == nil {
        return nil, &jrpc2.ErrorObject{
            Code:    jrpc2.InvalidParamsCode,
            Message: jrpc2.InvalidParamsMsg,
            Data:    "exactly two integers are required",
        }
    }

    return *p.X + *p.Y, nil
}

func main() {
    // create a new server instance
    s := jrpc2.NewServer(":8888", "/api/v1/rpc", nil)

    // register the add method
    s.Register("add", jrpc2.Method{Method: Add})

    // register the subtract method to proxy another rpc server
    // s.Register("add", jrpc2.Method{Url: "http://localhost:9999/api/v1/rpc"})

    // start the server instance
    s.Start()
}
```
When defining your own registered methods with the rpc server it is important to consider both named and positional parameters per the specification.

While named arguments are more straightforward, this library aims to be fully spec compliant, therefore positional parameters must be handled accordingly.

The ParseParams helper function should be used to ensure positional parameters are automatically resolved by the params struct's FromPositional handler method. The spec states *by-position: params MUST be an Array, containing the values in the Server expected order.*, so handling positional argument by direct subscript reference, where positional arguments are valid, should be considered safe.

### Multiplexing Server

The jrpc2 Server only support a single method handler.  This may not be suitable for versioned rpc APIs or any other implementation that requires more than a single rpc route.  The multiplexing server was added to support this use case.

Example: 

```golang
package main

import (
    "encoding/json"
    "github.com/bitwurx/jrpc2"
)

type AddV1Params struct {
    X int `json:x`
    Y int `json:y`
}

func (p *AddV1Params) FromPositional(params []interface{}) error {
    p.X = int(params[0].(float64))
    p.Y = int(params[1].(float64))
    return nil
}

func AddV1(params json.RawMessage) (interface{}, *jrpc2.ErrorObject) {
    p := new(AddV1Params)
    if err := jrpc2.ParseParams(params, p); err != nil {
        return nil, err
    }
    return p.X + p.Y, nil
}

type AddV2Params struct {
    Args []float64 `json:args`
}

func (p *AddV2Params) FromPositional(params []interface{}) error {
    p.Args = params[0].([]float64)
    return nil
}

func AddV2(params json.RawMessage) (interface{}, *jrpc2.ErrorObject) {
    p := new(AddV2Params)
    if err := jrpc2.ParseParams(params, p); err != nil {
        return nil, err
    }
    return p.Args[0] + p.Args[1], nil
}

func main() {
    v1 := jrpc2.NewMuxHandler()
    v1.Register("add", jrpc2.Method{Method: AddV1})
    v2 := jrpc2.NewMuxHandler()
    v2.Register("add", jrpc2.Method{Method: AddV2})
    s := jrpc2.NewMuxServer(":8080", nil)
    s.AddHandler("/rpc/v1", v1)
    s.AddHandler("/rpc/v2", v2)
    s.Start()
}
```

The mux server api is designed to mimic the single server api as closely as possible.  They key difference is the addition of the mux handler which handles method registration.  Once methods are registered to the handler, the handler is added to the mux server.  The mux server can then be started with the `Start()` method exactly like the single server.

Each registered handler isolates all registered methods so duplicating method names between handlers is fully supported.

*Warning: Mixing single and multiplexing servers can result in unexpected behavior and is not recommended.*

### Proxy Server

The jrpc2 HTTP server is capable of proxying another jrpc2 HTTP server's requests out of the box.  The `jrpc2.register` method allows rpc registration of a method.  Registration requires a method name and a url of the server to proxy.

The following request is an example of method registration:

```{"jsonrpc": "2.0", "method": "jrpc2.register", "params": ["subtract", "http://localhost:8080/api/v1/rpc"]}```

Methods can also be explicitly registered using the server's Register method:

```s.Register("add", jrpc2.Method{Url: "http://localhost:8080/api/v1/rpc"})```

### Running Tests

This library contains a set of api tests to verify spec compliance. The provided tests are a subset of the [Section 7 Examples](http://www.jsonrpc.org/specification#examples) here.

```sh
go test ./... -v
```

### License

```
Copyright (c) 2017 Jared Patrick <jared.patrick@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
