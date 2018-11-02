// Copyright (c) 2017 Jared Patrick <jared.patrick@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package jrpc2

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Error codes
const (
	ParseErrorCode     ErrorCode = -32700
	InvalidRequestCode ErrorCode = -32600
	MethodNotFoundCode ErrorCode = -32601
	InvalidParamsCode  ErrorCode = -32602
	InternalErrorCode  ErrorCode = -32603
	MethodExistsCode   ErrorCode = -32000
	URLSchemeErrorCode ErrorCode = -32001
)

// Error message
const (
	ParseErrorMsg     ErrorMsg = "Parse error"
	InvalidRequestMsg ErrorMsg = "Invalid Request"
	MethodNotFoundMsg ErrorMsg = "Method not found"
	InvalidParamsMsg  ErrorMsg = "Invalid params"
	InternalErrorMsg  ErrorMsg = "Internal error"
	ServerErrorMsg    ErrorMsg = "Server error"
	MethodExistsMsg   ErrorMsg = "Method exists"
	URLSchemeErrorMsg ErrorMsg = "URL scheme error"
)

// ErrorCode represents JRPC2 error code
type ErrorCode int

// ErrorMsg represents JRPC2 error message
type ErrorMsg string

// ErrorObject represents a response error object.
type ErrorObject struct {
	// Code indicates the error type that occurred.
	// Message provides a short description of the error.
	// Data is a primitive or structured value that contains additional information.
	// about the error.
	Code    ErrorCode   `json:"code"`
	Message ErrorMsg    `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// RequestObject represents a request object
type RequestObject struct {
	// Jsonrpc specifies the version of the JSON-RPC protocol.
	// Must be exactly "2.0".
	// Method contains the name of the method to be invoked.
	// Params is a structured value that holds the parameter values to be used during
	// the invocation of the method.
	// ID is a unique identifier established by the client.
	Jsonrpc string          `json:"jsonrpc"`
	Method  interface{}     `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// ResponseObject represents a response object.
type ResponseObject struct {
	// Jsonrpc specifies the version of the JSON-RPC protocol.
	// Must be exactly "2.0".
	// Error contains the error object if an error occurred while processing the request.
	// Result contains the result of the called method.
	// ID contains the client established request id or null.
	Jsonrpc string       `json:"jsonrpc"`
	Error   *ErrorObject `json:"error,omitempty"`
	Result  interface{}  `json:"result,omitempty"`
	ID      interface{}  `json:"id"`
}

// Params defines methods for processing request parameters.
type Params interface {
	FromPositional([]interface{}) error
}

// ParseParams processes the params data structure from the request.
// Named parameters will be umarshaled into the provided Params inteface.
// Positional arguments will be passed to Params interface's FromPositional method for
// extraction.
func ParseParams(params json.RawMessage, p Params) *ErrorObject {
	if err := json.Unmarshal(params, p); err != nil {
		errObj := &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMsg,
		}
		posParams := make([]interface{}, 0)
		if err = json.Unmarshal(params, &posParams); err != nil {
			errObj.Data = err.Error()
			return errObj
		}

		if err = p.FromPositional(posParams); err != nil {
			errObj.Data = err.Error()
			return errObj
		}
	}

	return nil
}

// NewResponse creates a bytes encoded representation of a response.
// Both result and error response objects can be created.
// The nl flag specifies if the response should be newline terminated.
func NewResponse(result interface{}, errObj *ErrorObject, id interface{}, nl bool) []byte {
	var resp bytes.Buffer
	body, _ := json.Marshal(&ResponseObject{
		Jsonrpc: "2.0",
		Error:   errObj,
		Result:  result,
		ID:      id,
	})
	resp.Write(body) // nolint: gosec

	if nl {
		resp.WriteString("\n") // nolint: gosec
	}

	return resp.Bytes()
}

// Batch is a wrapper around multiple response objects.
type Batch struct {
	// Responses contains the byte representations of a batch of responses.
	Responses [][]byte
}

// AddResponse inserts the response into the batch responses.
func (b *Batch) AddResponse(resp []byte) {
	b.Responses = append(b.Responses, resp)
}

// MakeResponse creates a bytes encoded representation of a response object.
func (b *Batch) MakeResponse() []byte {
	var resp bytes.Buffer
	resp.WriteString("[") // nolint: gosec

	for i, body := range b.Responses {
		resp.Write(body) // nolint: gosec
		if i < len(b.Responses)-1 {
			resp.WriteString(",") // nolint: gosec
		}
	}

	resp.WriteString("]\n") // nolint: gosec

	return resp.Bytes()
}

// Method represents an rpc method.
type Method struct {
	// URL is the url of the server that handles the method.
	// Method is the callable function
	URL    string
	Method func(params json.RawMessage) (interface{}, *ErrorObject)
}

// Server represents a jsonrpc 2.0 capable web server.
type Server struct {
	// Host is the host:port of the server.
	// Route is the path to the rpc api.
	// Methods contains the mapping of registered methods.
	// Headers contains response headers.
	Host    string
	Route   string
	Methods map[string]Method
	Headers map[string]string
}

// rpcHandler handles incoming rpc client requests.
func (s *Server) rpcHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.ParseRequest(w, r); err != nil {
		w.Write(NewResponse(nil, err, nil, true)) // nolint: errcheck, gosec
		return
	}
}

// HandleRequest validates, calls, and returns the result of a single rpc client request.
func (s *Server) HandleRequest(w http.ResponseWriter, req *RequestObject) {
	w.Header().Set("Content-Type", "application/json")
	for header, value := range s.Headers {
		w.Header().Set(header, value)
	}
	if err := s.ValidateRequest(req); err != nil {
		w.Write(NewResponse(nil, err, req.ID, true)) // nolint: errcheck, gosec
		return
	}

	if result, err := s.Call(req.Method, req.Params); err != nil {
		w.Write(NewResponse(nil, err, req.ID, true)) // nolint: errcheck, gosec
		return
	} else if req.ID != nil {
		w.Write(NewResponse(result, nil, req.ID, true)) // nolint: errcheck, gosec
	}
}

// HandleBatch validates, calls, and returns the results of a batch of rpc client requests.
// Batch methods are called in individual goroutines and collected in a single response.
func (s *Server) HandleBatch(w http.ResponseWriter, reqs []*RequestObject) {
	w.Header().Set("Content-Type", "application/json")
	if len(reqs) < 1 {
		err := &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMsg,
			Data:    `Batch must contain at least one request`,
		}
		w.Write(NewResponse(nil, err, nil, true)) // nolint: errcheck, gosec
	}

	var wg sync.WaitGroup
	batch := new(Batch)

	for _, req := range reqs {
		if err := s.ValidateRequest(req); err != nil {
			batch.AddResponse(NewResponse(nil, err, req.ID, false))
			continue
		}

		wg.Add(1)
		go func(req *RequestObject) {
			defer wg.Done()
			if result, err := s.Call(req.Method, req.Params); err != nil {
				batch.AddResponse(NewResponse(nil, err, req.ID, false))
			} else if req.ID != nil {
				batch.AddResponse(NewResponse(result, nil, req.ID, false))
			}
		}(req)
	}

	wg.Wait()
	if len(batch.Responses) > 0 {
		w.Write(batch.MakeResponse()) // nolint: errcheck, gosec
	}
}

// RegisterRPCParams is a paramater spec for the RegisterRPC method.
type RegisterRPCParams struct {
	// Name is the the name of the method being registered.
	// URL is the url of the server that handles the method.
	Name *string
	URL  *string
}

// FromPositional extracts the positional name and url parameters from a list of
// parameters.
func (rp *RegisterRPCParams) FromPositional(params []interface{}) error {
	if len(params) != 2 {
		return errors.New("register requires name and url parameters")
	}

	name := params[0].(string)
	url := params[1].(string)
	rp.Name = &name
	rp.URL = &url

	return nil
}

// RegisterRPC accepts a method name and server url to register a proxy rpc method.
// A method name can be only be registered once.
func (s *Server) RegisterRPC(params json.RawMessage) (interface{}, *ErrorObject) {
	p := new(RegisterRPCParams)

	if err := ParseParams(params, p); err != nil {
		return nil, err
	}

	if !strings.HasPrefix(*p.URL, "http://") && !strings.HasPrefix(*p.URL, "https://") {
		return nil, &ErrorObject{
			Code:    URLSchemeErrorCode,
			Message: URLSchemeErrorMsg,
			Data:    "url scheme must match http?s://",
		}
	}

	if _, ok := s.Methods[*p.Name]; ok {
		return nil, &ErrorObject{
			Code:    MethodExistsCode,
			Message: MethodExistsMsg,
		}
	}

	s.Methods[*p.Name] = Method{URL: *p.URL}

	return "success", nil
}

// Register maps the provided method to the given name for later method calls.
func (s *Server) Register(name string, method Method) {
	s.Methods[name] = method
}

// ParseRequest parses the json request body and unpacks into one or more.
// RequestObjects for single or batch processing.
func (s *Server) ParseRequest(w http.ResponseWriter, r *http.Request) *ErrorObject {
	var errObj *ErrorObject
	req := new(RequestObject)

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMsg,
			Data:    err.Error(),
		}
	}

	if err := json.Unmarshal(data, req); err != nil {
		errObj = &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMsg,
			Data:    err.Error(),
		}
	} else {
		s.HandleRequest(w, req)
	}

	if errObj != nil {
		var reqs []*RequestObject
		if err := json.Unmarshal(data, &reqs); err != nil {
			errObj = &ErrorObject{
				Code:    ParseErrorCode,
				Message: ParseErrorMsg,
				Data:    err.Error(),
			}
			return errObj
		}

		errObj = nil
		s.HandleBatch(w, reqs)
	}

	return errObj
}

// ValidateRequest validates that the request json contains valid values.
func (s *Server) ValidateRequest(req *RequestObject) *ErrorObject {
	if req.Jsonrpc != "2.0" {
		return &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMsg,
			Data:    `jsonrpc request member must be exactly '2.0'`,
		}
	}

	if _, ok := req.Method.(string); !ok {
		return &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMsg,
			Data:    "method name must be a string",
		}
	}

	if strings.HasPrefix(req.Method.(string), "rpc.") {
		return &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMsg,
			Data:    "method cannot match the pattern rpc.*",
		}
	}

	return nil
}

// Call invokes the named method with the provided parameters.
// If a method from the server Methods has a Method member will be called locally.
// If a method from the server Methods has a URL member it will be called by proxy.
func (s *Server) Call(name interface{}, params json.RawMessage) (interface{}, *ErrorObject) {
	method, ok := s.Methods[name.(string)]
	if !ok {
		return nil, &ErrorObject{
			Code:    MethodNotFoundCode,
			Message: MethodNotFoundMsg,
		}
	}
	if method.Method != nil {
		return method.Method(params)
	}
	if method.URL != "" {
		req := &RequestObject{
			Jsonrpc: "2.0",
			Method:  name,
			Params:  params,
			ID:      "1",
		}
		body, err := json.Marshal(req)
		if err != nil {
			return nil, &ErrorObject{
				Code:    InternalErrorCode,
				Message: InternalErrorMsg,
				Data:    err.Error(),
			}
		}
		data, err := http.Post(method.URL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return nil, &ErrorObject{
				Code:    InternalErrorCode,
				Message: InternalErrorMsg,
				Data:    err.Error(),
			}
		}

		resp := new(ResponseObject)
		rdr := bufio.NewReader(data.Body)
		dec := json.NewDecoder(rdr)
		err = dec.Decode(&resp)
		if err != nil {
			return nil, &ErrorObject{
				Code:    InternalErrorCode,
				Message: InternalErrorMsg,
				Data:    err.Error(),
			}
		}

		if resp.Result != nil {
			return resp.Result, nil
		} else if resp.Error != nil {
			return nil, resp.Error
		}
	}

	return nil, &ErrorObject{
		Code:    InternalErrorCode,
		Message: InternalErrorMsg,
		Data:    "Unable to call provided method",
	}
}

// Start binds the rpcHandler to the server route and starts the http server
func (s *Server) Start() {
	http.HandleFunc(s.Route, s.rpcHandler)
	log.Println(fmt.Sprintf("Starting server on %s at %s", s.Host, s.Route))
	log.Fatal(http.ListenAndServe(s.Host, nil))
}

// NewServer creates a new server instance
func NewServer(host, route string, headers map[string]string) *Server {
	s := &Server{
		Host:    host,
		Route:   route,
		Methods: make(map[string]Method),
		Headers: headers,
	}

	s.Methods["jrpc2.register"] = Method{Method: s.RegisterRPC}

	return s
}
