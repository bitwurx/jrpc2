Golang JSON-RPC 2.0 Server
===

This library is a server implementation of the [JSON-RPC 2.0 Specification](http://www.jsonrpc.org/specification). The library is fully spec compliant with support for named and positional arguments and batch requests.

*Example Usage:*

```
package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
		return errors.New(fmt.Sprintf("exactly two integers are required"))
	}

	x := params[0].(float64)
	y := params[1].(float64)
	ap.X = &x
	ap.Y = &y

	return nil
}

// Each method should match the prototype <fn(json.RawMessage) (inteface{}, *ErrorObject)>
func Add(params json.RawMessage) (interface{}, *ErrorObject) {
	p := new(AddParams)

	// ParseParams is a helper function that automatically invokes the FromPositional
	// method on the params instance if required
	if err := ParseParams(params, p); err != nil {
		return nil, err
	}

	if p.X == nil || p.Y == nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMsg,
			Data:    "exactly two integers are required",
		}
	}

	return *p.X + *p.Y, nil
}

func main() {
	s := NewServer(":8888", "/api/v1/rpc") // pass the server host and rpc handler path
	s.Register("add", Add)                 // register the add method
	s.Start()                              // start the rpc server
}
```
When defining your own registered methods with the rpc server it is important to consider both named and positional parameters per the specification. 

While named arguments are more straightforward, this library aims to be fully spec compliant, therefore positional parameters must be handled accordingly. 

The ParseParams helper function should be used to ensure positional parameters are automatically resolved by the params struct's FromPositional handler method. The spec states *by-position: params MUST be an Array, containing the values in the Server expected order.*, so handling positional argument by direct subscript reference, where positional arguments are valid, should be considered safe.

Running Tests
---
This library contains a set of api tests to verify spec compliance. The provided tests are a subset of the [Section 7 Examples](http://www.jsonrpc.org/specification#examples) here.

go test ./... -v