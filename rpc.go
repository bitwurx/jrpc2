package jrpc2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
)

/*
  Specification URLs:
    - https://www.jsonrpc.org/specification
    - https://www.simple-is-better.org/json-rpc/transport_http.html
*/

// Call invokes the named method with the provided parameters
func (s *Service) Call(name string, params json.RawMessage) (interface{}, *ErrorObject) {

	// check that request method member is not rpc-internal method
	if strings.HasPrefix(strings.ToLower(name), "rpc.") {
		return nil, &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMessage,
			Data:    "method cannot match the pattern rpc.*",
		}
	}

	// lookup method inside methods map
	method, ok := s.Methods[name]
	if !ok {
		return nil, &ErrorObject{
			Code:    MethodNotFoundCode,
			Message: MethodNotFoundMessage,
		}
	}

	// noncallable named method
	if method.Method == nil {
		return nil, &ErrorObject{
			Code:    InternalErrorCode,
			Message: InternalErrorMessage,
			Data:    "unable to call provided method",
		}
	}

	return method.Method(params)
}

// Do parses the JSON request body and returns response object
func (s *Service) Do(w http.ResponseWriter, r *http.Request) *ResponseObject {

	var errObj *ErrorObject

	reqObj := new(RequestObject)
	respObj := new(ResponseObject)

	// set JSON-RPC response version
	respObj.Jsonrpc = JSONRPCVersion

	// set custom response headers
	for header, value := range s.Headers {
		w.Header().Set(header, value)
	}

	// set Response Content-Type header
	w.Header().Set("Content-Type", "application/json")

	// set default response status code
	respObj.HTTPResponseStatusCode = http.StatusOK

	// check request Method
	if r.Method != "POST" {
		respObj.Error = &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMessage,
			Data:    "request method must be of POST type",
		}

		// set Response status code to 405 (method not allowed)
		respObj.HTTPResponseStatusCode = http.StatusMethodNotAllowed

		// set Allow header
		w.Header().Set("Allow", "POST")

		return respObj
	}

	// check request Content-Type header
	if !strings.EqualFold(r.Header.Get("Content-Type"), "application/json") {
		respObj.Error = &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMessage,
			Data:    "Content-Type header must be set to 'application/json'",
		}

		// set Response status code to 415 (unsupported media type)
		respObj.HTTPResponseStatusCode = http.StatusUnsupportedMediaType

		return respObj
	}

	// check request Accept header
	if !strings.EqualFold(r.Header.Get("Accept"), "application/json") {
		respObj.Error = &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMessage,
			Data:    "Accept header must be set to 'application/json'",
		}

		// set Response status code to 406 (not acceptable)
		respObj.HTTPResponseStatusCode = http.StatusNotAcceptable

		return respObj
	}

	// read request body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respObj.Error = &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMessage,
			Data:    err.Error(),
		}

		return respObj
	}

	// decode request body
	err = json.Unmarshal(data, &reqObj)
	if err != nil {
		// prepare default error object
		respObj.Error = &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMessage,
			Data:    err.Error(),
		}

		// additional error parsing
		switch err.(type) {
		// wrong data type data in request
		case *json.UnmarshalTypeError:
			// description of JSON value - "bool", "array", "number -5"
			switch err.(*json.UnmarshalTypeError).Value {
			// array data, batch request
			case "array":
				respObj.Error = &ErrorObject{
					Code:    NotImplementedCode,
					Message: NotImplementedMessage,
					Data:    "batch requests not supported",
				}
				return respObj
			}
			// name of the field holding the Go value
			switch err.(*json.UnmarshalTypeError).Field {
			// invalid data type for method
			case "method":
				respObj.Error = &ErrorObject{
					Code:    InvalidMethodCode,
					Message: InvalidMethodMessage,
					Data:    "method data type must be string",
				}
				return respObj
			}
		}

		return respObj
	}

	// validate JSON-RPC 2.0 request version member
	if reqObj.Jsonrpc != JSONRPCVersion {
		respObj.Error = &ErrorObject{
			Code:    InvalidRequestCode,
			Message: InvalidRequestMessage,
			Data:    fmt.Sprintf("jsonrpc request member must be exactly '%s'", JSONRPCVersion),
		}

		return respObj
	}

	switch reqObj.ID.(type) {
	case float64: // json package will assume float64 data type when you Unmarshal into an interface{}
		// truncate non integer part from float64
		if math.Trunc(reqObj.ID.(float64)) != reqObj.ID.(float64) {
			respObj.Error = &ErrorObject{
				Code:    InvalidIDCode,
				Message: InvalidIDMessage,
				Data:    "ID must be one of string, number or undefined",
			}
			return respObj
		}
	case string:
		// nothing to do here
	case nil:
		// nothing to do here
	default:
		respObj.Error = &ErrorObject{
			Code:    InvalidIDCode,
			Message: InvalidIDMessage,
			Data:    "ID must be one of string, number or undefined",
		}
		return respObj
	}

	// set response ID
	respObj.ID = &reqObj.ID

	// set notification flag
	if reqObj.ID == nil {
		reqObj.IsNotification = true
		respObj.IsNotification = true
	}

	// invoke named method with the provided parameters
	respObj.Result, errObj = s.Call(reqObj.Method, reqObj.Params)
	if errObj != nil {
		respObj.Error = errObj

		return respObj
	}

	return respObj
}

// RPCHandler handles incoming RPC client requests, generates responses
func (s *Service) RPCHandler(w http.ResponseWriter, r *http.Request) {

	// get response struct
	respObj := s.Do(w, r)

	// notification does not send responses to client
	if respObj.IsNotification {
		// set response header to 204, (no content)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// create a bytes encoded representation of a response struct
	resp, err := json.Marshal(respObj)
	if err != nil {
		panic(err.Error())
	}

	// write response code to HTTP writer interface
	w.WriteHeader(respObj.HTTPResponseStatusCode)

	// write data to HTTP writer interface
	_, err = w.Write(resp)
	if err != nil {
		panic(err.Error())
	}
}
