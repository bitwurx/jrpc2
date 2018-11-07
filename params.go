package jrpc2

import "encoding/json"

// GetPositionalFloat64Params parses positional param member of JSON-RPC 2.0 request
// that is know to contain float64 array
func GetPositionalFloat64Params(paramsRaw json.RawMessage) ([]float64, *ErrorObject) {

	params := make([]float64, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}

// GetPositionalInt64Params parses positional param member of JSON-RPC 2.0 request
// that is know to contain int64 array
func GetPositionalInt64Params(paramsRaw json.RawMessage) ([]int64, *ErrorObject) {

	params := make([]int64, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}

// GetPositionalIntParams parses positional param member of JSON-RPC 2.0 request
// that is know to contain int array
func GetPositionalIntParams(paramsRaw json.RawMessage) ([]int, *ErrorObject) {

	params := make([]int, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}

// GetPositionalUint64Params parses positional param member of JSON-RPC 2.0 request
// that is know to contain int64 array
func GetPositionalUint64Params(paramsRaw json.RawMessage) ([]uint64, *ErrorObject) {

	params := make([]uint64, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}

// GetPositionalUintParams parses positional param member of JSON-RPC 2.0 request
// that is know to contain uint array
func GetPositionalUintParams(paramsRaw json.RawMessage) ([]uint, *ErrorObject) {

	params := make([]uint, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}

// GetPositionalStringParams parses positional param member of JSON-RPC 2.0 request
// that is know to contain string array
func GetPositionalStringParams(paramsRaw json.RawMessage) ([]string, *ErrorObject) {

	params := make([]string, 0)

	err := json.Unmarshal(paramsRaw, &params)
	if err != nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMessage,
			Data:    err.Error(),
		}
	}

	return params, nil
}
