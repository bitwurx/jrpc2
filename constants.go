package jrpc2

// JSONRPCVersion specifies the version of the JSON-RPC protocol
const JSONRPCVersion string = "2.0"

// Error codes
const (
	ParseErrorCode     int = -32700
	InvalidRequestCode int = -32600
	MethodNotFoundCode int = -32601
	InvalidParamsCode  int = -32602
	InternalErrorCode  int = -32603
	NotImplementedCode int = -32000
	InvalidIDCode      int = -32001
	InvalidMethodCode  int = -32002
)

// Error message
const (
	ParseErrorMessage     string = "Parse error"
	InvalidRequestMessage string = "Invalid Request"
	MethodNotFoundMessage string = "Method not found"
	InvalidParamsMessage  string = "Invalid params"
	InternalErrorMessage  string = "Internal error"
	NotImplementedMessage string = "Not implemented"
	InvalidIDMessage      string = "Invalid ID"
	InvalidMethodMessage  string = "Invalid method"
)
