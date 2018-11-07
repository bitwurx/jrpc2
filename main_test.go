package jrpc2

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const endpoint = "jrpc"
const port = "31500"

var id, x, y int
var r *strings.Replacer

type SubtractParams struct {
	X *float64 `json:"X"`
	Y *float64 `json:"Y"`
}

type Result struct {
	Jsonrpc string       `json:"jsonrpc"`
	Error   *ErrorObject `json:"error"`
	Result  interface{}  `json:"result"`
	ID      interface{}  `json:"id"`
}

func Subtract(paramsRaw json.RawMessage) (interface{}, *ErrorObject) {

	paramObj := new(SubtractParams)

	err := json.Unmarshal(paramsRaw, paramObj)
	if err != nil {

		errObj := &ErrorObject{
			Code:    ParseErrorCode,
			Message: ParseErrorMessage,
			Data:    err.Error(),
		}

		switch err.(type) {
		case *json.UnmarshalTypeError:
			switch err.(*json.UnmarshalTypeError).Value {
			case "array":

				var params []float64

				params, errObj = GetPositionalFloat64Params(paramsRaw)
				if errObj != nil {
					return nil, errObj
				}

				if len(params) != 2 {
					return nil, &ErrorObject{
						Code:    InvalidParamsCode,
						Message: InvalidParamsMessage,
						Data:    "exactly two integers are required",
					}
				}

				paramObj.X = &params[0]
				paramObj.Y = &params[1]

			default:
				return nil, errObj
			}
		default:
			return nil, errObj
		}
	}

	if *paramObj.X == 999.0 && *paramObj.Y == 999.0 {
		return nil, &ErrorObject{
			Code:    -320099,
			Message: "Custom error",
			Data:    "mock server error",
		}
	}

	return *paramObj.X - *paramObj.Y, nil
}

// nolint: gochecknoinits
func init() {

	// Seed random
	rand.Seed(time.Now().UnixNano())

	// RequestID for tests
	id = rand.Intn(42)

	// X variable for subtract method
	x = rand.Intn(60)

	// Y variable for subtract method
	y = rand.Intn(30)

	// Replacer for request data
	r = strings.NewReplacer(
		"#ID", strconv.Itoa(id),
		"#X", strconv.Itoa(x),
		"#Y", strconv.Itoa(y),
	)

	go func() {
		s := Create(
			net.JoinHostPort("localhost", port),
			fmt.Sprintf("/%s", endpoint),
			map[string]string{
				"Server":                        "JSON-RPC/2.0 (Golang)",
				"Access-Control-Allow-Origin":   "*",
				"Access-Control-Expose-Headers": "Content-Type",
				"Access-Control-Allow-Methods":  "POST",
				"Access-Control-Allow-Headers":  "Content-Type",
			},
		)

		s.Register("update", Method{Method: func(params json.RawMessage) (interface{}, *ErrorObject) { return nil, nil }})
		s.Register("subtract", Method{Method: Subtract})

		s.Start()
	}()
}

// Wrapper for sending request to mock server
func sendTestRequest(request string) (*http.Response, error) {

	// full JSON-RPC 2.0 URL
	url := fmt.Sprintf("http://%s/%s", net.JoinHostPort("localhost", port), endpoint)

	headers := map[string]string{
		"Accept":       "application/json", // set Accept header
		"Content-Type": "application/json", // set Content-Type header
	}

	// call wrapper
	return httpPost(url, request, headers)
}

// Generic wrapper for HTTP POST
func httpPost(url, request string, headers map[string]string) (*http.Response, error) {

	// request data
	buf := bytes.NewBuffer([]byte(r.Replace(request)))

	// prepare default http client config
	client := &http.Client{}

	// set request type to POST
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}

	// setting specified headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// send request
	return client.Do(req)
}

func TestNonPOSTRequestType(t *testing.T) {

	var result Result

	// full JSON-RPC 2.0 URL
	url := fmt.Sprintf("http://%s/%s", net.JoinHostPort("localhost", port), endpoint)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected HTTP status code to be %d", http.StatusMethodNotAllowed)
	}

	if v := resp.Header.Get("Allow"); v != "POST" {
		t.Fatal("got unexpected Server value")
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != InvalidRequestCode {
		t.Fatalf("expected error code to be %d", InvalidRequestCode)
	}
	if result.Error.Message != InvalidRequestMessage {
		t.Fatalf("expected error message to be '%s'", InvalidRequestMessage)
	}
}

func TestRequestHeaderWrongContentType(t *testing.T) {

	var result Result

	// full JSON-RPC 2.0 URL
	url := fmt.Sprintf("http://%s/%s", net.JoinHostPort("localhost", port), endpoint)

	headers := map[string]string{
		"Accept":       "application/json",      // set Accept header
		"Content-Type": "x-www-form-urlencoded", // set Content-Type header
	}

	// call wrapper
	resp, err := httpPost(url, `{}`, headers)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected HTTP status code to be %d", http.StatusUnsupportedMediaType)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != ParseErrorCode {
		t.Fatalf("expected error code to be %d", ParseErrorCode)
	}
	if result.Error.Message != ParseErrorMessage {
		t.Fatalf("expected error message to be '%s'", ParseErrorMessage)
	}
}

func TestRequestHeaderWrongAccept(t *testing.T) {

	var result Result

	// full JSON-RPC 2.0 URL
	url := fmt.Sprintf("http://%s/%s", net.JoinHostPort("localhost", port), endpoint)

	headers := map[string]string{
		"Accept":       "x-www-form-urlencoded", // set Accept header
		"Content-Type": "application/json",      // set Content-Type header
	}

	// call wrapper
	resp, err := httpPost(url, `{}`, headers)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusNotAcceptable {
		t.Fatalf("expected HTTP status code to be %d", http.StatusNotAcceptable)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != ParseErrorCode {
		t.Fatalf("expected error code to be %d", ParseErrorCode)
	}
	if result.Error.Message != ParseErrorMessage {
		t.Fatalf("expected error message to be '%s'", ParseErrorMessage)
	}
}

func TestResponseHeaders(t *testing.T) {

	resp, err := sendTestRequest(`{}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	if v := resp.Header.Get("Server"); v != "JSON-RPC/2.0 (Golang)" {
		t.Fatal("got unexpected Server value")
	}
}
func TestIDStringType(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "update", "id": "ID:42"}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(string) != "ID:42" {
		t.Fatal("expected ID to be ID:42")
	}
	if result.Error != nil {
		t.Fatal("expected error to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
}

func TestIDNumberType(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "update", "id": 42}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(float64) != float64(42) {
		t.Fatal("expected ID to be 42")
	}
	if result.Error != nil {
		t.Fatal("expected error to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
}

func TestIDTypeError(t *testing.T) {

	reqList := make([]string, 0)
	reqList = append(
		reqList,
		`{"jsonrpc": "2.0", "method": "update", "id": 42.42}`,         // float
		`{"jsonrpc": "2.0", "method": "update", "id": [42, 42]}`,      // array
		`{"jsonrpc": "2.0", "method": "update", "id": {"value": 42}}`, // object
	)

	for _, el := range reqList {

		var result Result

		resp, err := sendTestRequest(el)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
		}

		err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
		if err != nil {
			t.Fatal(err)
		}

		if result.Jsonrpc != JSONRPCVersion {
			t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
		}
		if result.ID != nil {
			t.Fatal("expected ID to be nil")
		}
		if result.Result != nil {
			t.Fatal("expected result to be nil")
		}
		if result.Error == nil {
			t.Fatal("expected error to be not nil")
		}
		if result.Error.Code != InvalidIDCode {
			t.Fatalf("expected error code to be %d", InvalidIDCode)
		}
		if result.Error.Message != InvalidIDMessage {
			t.Fatalf("expected error message to be '%s'", InvalidIDMessage)
		}
	}
}
func TestNonExistentMethod(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "foobar", "id": #ID}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	_ = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(float64) != float64(id) {
		t.Fatalf("expected ID to be %d", id)
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != MethodNotFoundCode {
		t.Fatalf("expected error code to be %d", MethodNotFoundCode)
	}
	if result.Error.Message != MethodNotFoundMessage {
		t.Fatalf("expected error message to be '%s'", MethodNotFoundMessage)
	}
}

func TestInvalidMethodObjectType(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": 1, "params": "bar", "id": #ID}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != InvalidMethodCode {
		t.Fatalf("expected error code to be %d", InvalidMethodCode)
	}
	if result.Error.Message != InvalidMethodMessage {
		t.Fatalf("expected error message to be '%s'", InvalidMethodMessage)
	}
}
func TestNamedParameters(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "subtract", "params": {"X": #X, "Y": #Y}, "id": #ID}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(float64) != float64(id) {
		t.Fatalf("expected ID to be %d", id)
	}
	if result.Error != nil {
		t.Fatal("expected error to be nil")
	}
	if result.Result == nil {
		t.Fatal("expected result to be not nil")
	}
	if result.Result.(float64) != float64(x-y) {
		t.Fatalf("expected result to be %f", float64(x-y))
	}
}
func TestNotification(t *testing.T) {

	req := `{"jsonrpc": "2.0", "method": "subtract", "params": {"X": #X, "Y": #Y}}`
	reqList := make([]string, 0)
	reqList = append(reqList, r.Replace(req), `{"jsonrpc": "2.0", "method": "update"}`)

	for _, el := range reqList {

		var result Result

		resp, err := sendTestRequest(el)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			err = resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
		}()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected HTTP status code to be %d", http.StatusNoContent)
		}

		err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
		if err != io.EOF {
			t.Fatal("expected empty response to notification request")
		}
	}
}

func TestBatchNotifications(t *testing.T) {

	var result Result

	req := `[
			{"jsonrpc": "2.0", "method": "subtract", "params": {"X": #X, "Y": #Y}},
			{"jsonrpc": "2.0", "method": "subtract", "params": {"X": #Y, "Y": #X}}
		]`

	resp, err := sendTestRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatalf("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != NotImplementedCode {
		t.Fatalf("expected error code to be %d", NotImplementedCode)
	}
	if result.Error.Message != NotImplementedMessage {
		t.Fatalf("expected error message to be '%s'", NotImplementedMessage)
	}
}
func TestPositionalParamters(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "subtract", "params": [#X, #Y], "id": #ID}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(float64) != float64(id) {
		t.Fatalf("expected ID to be %d", id)
	}
	if result.Error != nil {
		t.Fatal("expected error to be nil")
	}
	if result.Result == nil {
		t.Fatal("expected result to be not nil")
	}
	if result.Result.(float64) != float64(x-y) {
		t.Fatalf("expected result to be %f", float64(x-y))
	}
}

func TestPositionalParamtersError(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "subtract", "params": [999, 999], "id": #ID}`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID.(float64) != float64(id) {
		t.Fatalf("expected ID to be %d", id)
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != -320099 {
		t.Fatal("expected code to be -320099")
	}
	if result.Error.Message != "Custom error" {
		t.Fatal("expected message to be 'Custom error'")
	}
	if result.Error.Data != "mock server error" {
		t.Fatal("expected data to be 'mock server error'")
	}
}
func TestInvalidJSON(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`{"jsonrpc": "2.0", "method": "update, "params": "bar", "baz]`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != ParseErrorCode {
		t.Fatalf("expected error code to be %d", ParseErrorCode)
	}
	if result.Error.Message != ParseErrorMessage {
		t.Fatalf("expected error message to be '%s'", ParseErrorMessage)
	}
}

func TestBatchInvalidJSON(t *testing.T) {

	var result Result

	req := `[
			{"jsonrpc": "2.0", "method": "subtract", "params": {"X": 42, "Y": 23}, "id": "1"},
			{"jsonrpc": "2.0", "method"
		]`

	resp, err := sendTestRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatal("expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != ParseErrorCode {
		t.Fatalf("expected error code to be %d", ParseErrorCode)
	}
	if result.Error.Message != ParseErrorMessage {
		t.Fatalf("expected error message to be '%s'", ParseErrorMessage)
	}
}

func TestBatchEmptyArray(t *testing.T) {

	var result Result

	resp, err := sendTestRequest(`[]`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result.Jsonrpc != JSONRPCVersion {
		t.Fatalf("expected Jsonrpc to be '%s'", JSONRPCVersion)
	}
	if result.ID != nil {
		t.Fatal("expected ID to be nil")
	}
	if result.Result != nil {
		t.Fatalf("Expected result to be nil")
	}
	if result.Error == nil {
		t.Fatal("expected error to be not nil")
	}
	if result.Error.Code != NotImplementedCode {
		t.Fatalf("expected error code to be %d", NotImplementedCode)
	}
	if result.Error.Message != NotImplementedMessage {
		t.Fatalf("expected error message to be '%s'", NotImplementedMessage)
	}
	if result.Error.Data != "batch requests not supported" {
		t.Fatal("expected data to be 'batch requests not supported'")
	}
}

func TestInvalidBatch(t *testing.T) {

	var results []Result

	resp, err := sendTestRequest(`[4, 2]`)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code to be %d", http.StatusOK)
	}

	err = json.NewDecoder(bufio.NewReader(resp.Body)).Decode(&results)
	if err == nil {
		t.Fatal("expected decoder error for batch request")
	}
}
