package jrpc2

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

type SumParams struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

func (ap *SumParams) FromPositional(params []interface{}) error {
	if len(params) != 2 {
		return fmt.Errorf("exactly two integers are required")
	}

	x := params[0].(float64)
	y := params[1].(float64)
	ap.X = &x
	ap.Y = &y

	return nil
}

func Sum(params json.RawMessage) (interface{}, *ErrorObject) {
	p := new(SumParams)

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

type SubtractParams struct {
	X *float64 `json:"minuend"`
	Y *float64 `json:"subtrahend"`
}

func (ap *SubtractParams) FromPositional(params []interface{}) error {
	if len(params) != 2 {
		return fmt.Errorf("exactly two integers are required")
	}

	x := params[0].(float64)
	y := params[1].(float64)
	ap.X = &x
	ap.Y = &y

	return nil
}

func Subtract(params json.RawMessage) (interface{}, *ErrorObject) {
	p := new(SubtractParams)

	if err := ParseParams(params, p); err != nil {
		return nil, err
	}

	if *p.X == 999.0 && *p.Y == 999.0 {
		return nil, &ErrorObject{
			Code:    -32001,
			Message: ServerErrorMsg,
			Data:    "Mock error",
		}
	}

	if p.X == nil || p.Y == nil {
		return nil, &ErrorObject{
			Code:    InvalidParamsCode,
			Message: InvalidParamsMsg,
			Data:    "exactly two integers are required",
		}
	}

	return *p.X - *p.Y, nil
}

// nolint: gochecknoinits
func init() {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() { // subtract method remote server
		s := NewServer(":31501", "/api/v2/rpc", nil)
		s.Register("subtract", Method{Method: Subtract})
		s.Start()
	}()

	go func() { // primary server with subtract remote server proxy
		s := NewServer(":31500", "/api/v1/rpc", map[string]string{
			"X-Test-Header": "some-test-value",
		})
		s.Register("sum", Method{Method: Sum})
		s.Register("update", Method{Method: func(params json.RawMessage) (interface{}, *ErrorObject) { return nil, nil }})
		s.Register("foobar", Method{Method: func(params json.RawMessage) (interface{}, *ErrorObject) { return nil, nil }})
		s.Start()
	}()

	go func() {
		for {
			body := `{"jsonrpc": "2.0", "method": "jrpc2.register", "params": ["subtract", "http://localhost:31501/api/v2/rpc"]}`
			buf := bytes.NewBuffer([]byte(body))
			_, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		wg.Done()
	}()

	wg.Wait()
}

func TestResponseHeaders(t *testing.T) {
	buf := bytes.NewBuffer([]byte(`{}`))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	if v := resp.Header.Get("X-Test-Header"); v != "some-test-value" {
		t.Fatal("got unexpected X-Test-Header value")
	}
}

func TestRpcCallWithPositionalParamters(t *testing.T) {
	table := []struct {
		Body    string
		Jsonrpc string `json:"jsonrpc"`
		Result  int    `json:"result"`
		ID      int    `json:"id"`
	}{
		{`{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}`, "2.0", 19, 1},
		{`{"jsonrpc": "2.0", "method": "subtract", "params": [23, 42], "id": 2}`, "2.0", -19, 2},
	}

	for _, tc := range table {
		var result struct {
			Jsonrpc string      `json:"jsonrpc"`
			Result  int         `json:"result"`
			Error   interface{} `json:"error"`
			ID      int         `json:"id"`
		}

		buf := bytes.NewBuffer([]byte(tc.Body))
		resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		dec := json.NewDecoder(rdr)
		err = dec.Decode(&result)
		if err != nil {
			t.Fatal(err)
		}

		if result.Error != nil {
			t.Fatal("Expected error to be nil")
		}
		if result.Jsonrpc != tc.Jsonrpc {
			t.Fatal("Invalid jsonrpc member value")
		}
		if result.Result != tc.Result {
			t.Fatalf("Expected result to be %d", tc.Result)
		}
		if result.ID != tc.ID {
			t.Fatalf("Expected id to be %d", tc.ID)
		}
	}
}

func TestRpcCallWithPositionalParamtersError(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Result  interface{} `json:"result"`
		Err     ErrorObject `json:"error"`
		ID      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": "subtract", "params": [999, 999], "id": 1}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	err = dec.Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != nil {
		t.Fatal("Expected result to be nil")
	}
	if result.Err.Code != -32001 {
		t.Fatal("Expected code to be -32001")
	}
	if result.Err.Message != "Server error" {
		t.Fatal("Expected message to be 'Server error'")
	}
	if result.Err.Data != "Mock error" {
		t.Fatal("Expected data to be 'Mock error'")
	}
	if result.ID != 1 {
		t.Fatal("Expected id to be 1")
	}
}

func TestRpcCallWithNamedParameters(t *testing.T) {
	table := []struct {
		Body    string
		Jsonrpc string `json:"jsonrpc"`
		Result  int    `json:"result"`
		ID      int    `json:"id"`
	}{
		{`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}`, "2.0", 19, 3},
		{`{"jsonrpc": "2.0", "method": "subtract", "params": {"minuend": 42, "subtrahend": 23}, "id": 4}`, "2.0", 19, 4},
	}

	for _, tc := range table {
		var result struct {
			Jsonrpc string      `json:"jsonrpc"`
			Result  int         `json:"result"`
			Error   interface{} `json:"error"`
			ID      int         `json:"id"`
		}

		buf := bytes.NewBuffer([]byte(tc.Body))
		resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		dec := json.NewDecoder(rdr)
		err = dec.Decode(&result)
		if err != nil {
			t.Fatal(err)
		}

		if result.Error != nil {
			t.Fatal("Expected error to be nil")
		}
		if result.Jsonrpc != tc.Jsonrpc {
			t.Fatal("Invalid jsonrpc member value")
		}
		if result.Result != tc.Result {
			t.Fatalf("Expected result to be %d", tc.Result)
		}
		if result.ID != tc.ID {
			t.Fatalf("Expected id to be %d", tc.ID)
		}
	}
}

func TestNotification(t *testing.T) {
	table := []string{
		`{"jsonrpc": "2.0", "method": "update", "params": [1,2,3,4,5]}`,
		`{"jsonrpc": "2.0", "method": "foobar"}`,
	}

	for _, body := range table {
		buf := bytes.NewBuffer([]byte(body))
		resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		data, err := rdr.ReadBytes('\b')
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		if len(data) > 0 {
			t.Fatal("Expected notification to return no response body")
		}
	}
}

func TestCallOfNotExistentMethod(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": "fooba", "id": "1"}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	_ = dec.Decode(&result)

	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Err.Code != -32601 {
		t.Fatal("expected error code -32601")
	}
	if result.Err.Message != "Method not found" {
		t.Fatal("expected message to be 'Message not found'")
	}
}

func TestCallWithInvalidJSON(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	err = dec.Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Err.Code != -32700 {
		t.Fatal("expected error code -32700")
	}
	if result.Err.Message != "Parse error" {
		t.Fatal("expected message to be 'Parse error'")
	}
}

func TestCallWithInvalidRequestObject(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": 1, "params": "bar"}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	err = dec.Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Err.Code != -32600 {
		t.Fatal("expected error code -32600")
	}
	if result.Err.Message != "Invalid Request" {
		t.Fatal("expected message to be 'Invalid Request'")
	}
}

func TestBatchCallWithInvalidJSON(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `[
        {"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
        {"jsonrpc": "2.0", "method"
    ]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	err = dec.Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Err.Code != -32700 {
		t.Fatal("expected error code -32700")
	}
	if result.Err.Message != "Parse error" {
		t.Fatal("expected message to be 'Parse error'")
	}
}

func TestBatchCallWithAnEmptyArray(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `[]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	err = dec.Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Err.Code != -32600 {
		t.Fatal("expected error code -32600")
	}
	if result.Err.Message != "Invalid Request" {
		t.Fatal("expected message to be 'Invalid Request'")
	}
}

func TestCallWithAnInvalidBatch(t *testing.T) {
	var results []struct {
		Jsonrpc string      `json:"jsonrpc"`
		Err     ErrorObject `json:"error"`
		Result  interface{} `json:"result"`
		ID      int         `json:"id"`
	}

	body := `[]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	_ = dec.Decode(&results)

	for _, result := range results {
		if result.Result != nil {
			t.Fatal("expected result to be nil")
		}
		if result.Err.Code != -32600 {
			t.Fatal("expected error code -32600")
		}
		if result.Err.Message != "Invalid Request" {
			t.Fatal("expected message to be 'Invalid Request'")
		}
	}
}

func TestCallBatchWithNotifications(t *testing.T) {
	body := `[
        {"jsonrpc": "2.0", "method": "sum", "params": [1,2]},
        {"jsonrpc": "2.0", "method": "subtract", "params": [7,2]}
	]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	data, err := rdr.ReadBytes('\n')
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}

	if len(data) > 0 {
		t.Fatal("Expected batch notification to return no response body")
	}
}
