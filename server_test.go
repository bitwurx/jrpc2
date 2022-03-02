package jrpc2

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
		return errors.New(fmt.Sprintf("exactly two integers are required"))
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
		return errors.New(fmt.Sprintf("exactly two integers are required"))
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

type SayParams struct {
	Message string `json:"message"`
}

func (sp *SayParams) FromPositional(params []interface{}) error {
	if len(params) != 1 {
		return fmt.Errorf("exactly one argument is required")
	}
	sp.Message = params[0].(string)
	return nil
}

func Say(ctx context.Context, params json.RawMessage) (interface{}, *ErrorObject) {
	ruser := ctx.Value("user")
	var user string
	if ruser == nil {
		user = ""
	} else {
		user = ruser.(string)
	}
	p := new(SayParams)

	if err := ParseParams(params, p); err != nil {
		return nil, err
	}

	return fmt.Sprintf("%s %s!", p.Message, user), nil
}

func newTLSCert() (string, string) {
	tlsCert := []byte(`-----BEGIN CERTIFICATE-----
MIIDITCCAgmgAwIBAgIUMNBQr3kf3X1swwE6vbaNS74xQJ0wDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTIyMDMwMjIwNDMwOVoYDzIxMjIw
MjA2MjA0MzA5WjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDZxW11LNPTrhSBqjoXO0c4jFEJ669Js1UmQxgf9/5Q
M3s9kQrcQLOQM00LpcVS43ko1FANAwhMajp+wJjDY2t3eBQgS+XcqFv5HqQeo+Qw
cJVwvUm8sEF3qlHgtFm5Yei36hn0Id1oUTPJEjki3USmLmeIo7ofKqLuMG2+zsEq
9Krq9B+9BYYKtYFDZIBsEx9IZVFBuLXbhjTooCDDVNASNIgYG779OY6mTGv10QqM
zBC+U2wd0Vk3xhQ/3BKfC3t6GI6x/WSKGzmUe6LB5wfKubiCqfNEupt8+PdEhTAB
rNEluJeaPIWn9MS2GVm8D2l6Zi5zBK75+EQmh75ZelOlAgMBAAGjaTBnMB0GA1Ud
DgQWBBQByHuYCt/2rpJEHdH+M11cx5vsrjAfBgNVHSMEGDAWgBQByHuYCt/2rpJE
HdH+M11cx5vsrjAPBgNVHRMBAf8EBTADAQH/MBQGA1UdEQQNMAuCCWxvY2FsaG9z
dDANBgkqhkiG9w0BAQsFAAOCAQEAud/HFsPIyDjBROBBmn6YpKfoAOoUS20hf9nF
TAy7A8JbWx2V7ba13mjYOiutz6AsKpICc5pXdbt11dySajT3Vnts73RO9ajlIA6i
LQ4/0GW10V2gvNuCxZkCYZfKraRJx1uXFcZXCzMiLNtFnk07RtT/cT6ZzZi0oTSJ
zmf0Vx0rKjNgJPUZyvbn6XU4x5PAHuXKAzrrN44YPHlsuS7dblzqyp4VQ1UfCH4M
psmMk3xlbaBDSoH2WkPSLKIdqVJ6vgiD3JA1oeH4hgvNG4MdpF1zdDESy9ZmnfBs
Kx1Jw+4W8q9h5I15UDZVWByWk5LMysMoktewZuNW2qVQHvGCvQ==
-----END CERTIFICATE-----`)
	tlsCertFile, err := ioutil.TempFile("", "tls")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tlsCertFile.Write(tlsCert); err != nil {
		log.Fatal(err)
	}
	if err := tlsCertFile.Close(); err != nil {
		log.Fatal(err)
	}
	tlsKey := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDZxW11LNPTrhSB
qjoXO0c4jFEJ669Js1UmQxgf9/5QM3s9kQrcQLOQM00LpcVS43ko1FANAwhMajp+
wJjDY2t3eBQgS+XcqFv5HqQeo+QwcJVwvUm8sEF3qlHgtFm5Yei36hn0Id1oUTPJ
Ejki3USmLmeIo7ofKqLuMG2+zsEq9Krq9B+9BYYKtYFDZIBsEx9IZVFBuLXbhjTo
oCDDVNASNIgYG779OY6mTGv10QqMzBC+U2wd0Vk3xhQ/3BKfC3t6GI6x/WSKGzmU
e6LB5wfKubiCqfNEupt8+PdEhTABrNEluJeaPIWn9MS2GVm8D2l6Zi5zBK75+EQm
h75ZelOlAgMBAAECggEBAL4yr5nykAPGe8SP/3KA5IBgFPtcLFlrVog3e1+YgjZb
8FxiTKD3pZzhIX51xzTQ1eYyIMRsjJfpA7Pm1MV6FMdgSfu3Lkidhs66006rh8ZC
3lJ8EGXLbzJrwF1IR0EhYVcYEJjn5u+QVHFeCCcKKEYYK3bswMctvuXXyFIpVA8F
y384dc+mP8SG5n9YfjLj7cnGkNJNpfMjYWJsjH56bIdAMFk7XR6XTaoy+J6T/NhB
MSBo8ao36GfzQcbxCGUw+HVP/O6Jq7u3zcVej0iK3GY9Y6Jl4YhrdcnfwpfZzTTI
4fDxxxlZiieoso3mt2ym7wOLPjm4r4zycGTaZAj7pCECgYEA9E68IzgemfvkvmE/
p8wXErbGtsIUBLsy8dvW7TsFc6KuFab0vPyL8vIEmqNcnKqT1JL7vmb8dIsXkg2Y
WbSQ92n9WBRuJyIbr81XvV6IYsC1P9zO8AWIPt/PEd5nM0ylqn9RzMekiMzAeJKJ
qSlob3sQhrYXCxEK8c/8RgX64P0CgYEA5DGReLb+VWmZaOjUkBWLAjv4VioMCCJ/
VNMON2fPE7k95XaWdLMQvnP+q9A+sxJf6J3GYYU3RFv7WfTjvdNhjnphFxdTDpmd
CeeSuJdluZE9PbAs+b74jLR3nwNIWaR+J+PAcV46/4w9KlZ2GMaBKKbxpIyd8/W6
MNaB/AHwcckCgYATDIiS3m9UZlWhmoeSF9G8vc+ktGFHNSl1vkR13uI/7/FO8uOm
ULLA0KoXPKGd/ZblPkiuwezxUV8XHkRAyll7USJV2dH07y3leUdcFqDfwlLfleH0
yRmkfWLx67t0Poe0UZUZOH/VwtFHFXXyYK4p8xiIyG3niP6neCYdd53mKQKBgQCp
EwkD9iIvytQ95PVJ5IxglWqE/RZ5GIZbpR1Nc/78UC5KTDliMiLf2jYBu4QZTi39
vpj0PK4cWkK7/jSXu3z3AjnZ0BBcKvkuE4SkfJiEi9ZiVJyeVx71selHyjjbIoPO
rnMyDG2OVqwjKHjMFpgwNLGqB/4oehMAiI8613z98QKBgBx/QoSto+Yv4DiAvnAW
e3jH3CGpnfegQuIMQRoGsV3/DS/cN3Ey4GGGLfnxMoIFZaQenm+GtLS6Dfzc1w+u
5BHqEY8xPgX2YQQ/YWqKMDurt8rWIPS04GYYY3UK2m+2CGlTUuR2kN1yUn88i+PO
p2d2T8pkQLLdnOk/VdnNu9QR
-----END PRIVATE KEY-----`)
	tlsKeyFile, err := ioutil.TempFile("", "tls")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tlsKeyFile.Write(tlsKey); err != nil {
		log.Fatal(err)
	}
	if err := tlsKeyFile.Close(); err != nil {
		log.Fatal(err)
	}
	return tlsCertFile.Name(), tlsKeyFile.Name()
}

func init() {
	var wg sync.WaitGroup
	wg.Add(1)
	crt, key := newTLSCert()
	defer os.Remove(crt)
	defer os.Remove(key)
	log.SetOutput(ioutil.Discard)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	go func() { // subtract method remote server
		s := NewMuxServer(":31501", nil)
		h := NewMuxHandler()
		h.Register("subtract", Method{Method: Subtract})
		s.AddHandler("/api/v2/rpc", h)
		s.Start()
	}()

	go func() { // subtract method remote TLS server
		s := NewMuxServer(":31510", nil)
		h := NewMuxHandler()
		h.Register("subtract", Method{Method: Subtract})
		s.AddHandler("/api/v3/rpc", h)
		s.StartTLS(crt, key)
	}()

	go func() { // primary server with subtract remote server proxy
		s := NewServer(":31500", "/api/v1/rpc", map[string]string{
			"X-Test-Header": "some-test-value",
		})
		s.Register("sum", Method{Method: Sum})
		s.RegisterWithContext("say", MethodWithContext{Method: Say})
		s.StartWithMiddleware(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				req := r.WithContext(context.WithValue(r.Context(), "user", r.Header.Get("user")))
				next(w, req)
			}
		})
	}()

	go func() { // primary server with subtract remote TLS server proxy
		s := NewServer(":31511", "/api/v4/rpc", map[string]string{
			"X-Test-Header": "some-test-value",
		})
		s.Register("update", Method{Method: func(params json.RawMessage) (interface{}, *ErrorObject) { return nil, nil }})
		s.Register("foobar", Method{Method: func(params json.RawMessage) (interface{}, *ErrorObject) { return nil, nil }})
		s.StartTLSWithMiddleware(crt, key, func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				req := r.WithContext(context.WithValue(r.Context(), "user", r.Header.Get("user")))
				next(w, req)
			}
		})
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
			body2 := `{"jsonrpc": "2.0", "method": "jrpc2.register", "params": ["subtract", "http://localhost:31510/api/v3/rpc"]}`
			buf2 := bytes.NewBuffer([]byte(body2))
			_, err2 := http.Post("http://localhost:31511/api/v4/rpc", "application/json", buf2)
			if err2 != nil {
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
	defer resp.Body.Close()
	if v := resp.Header.Get("X-Test-Header"); v != "some-test-value" {
		t.Fatal("got unexpected X-Test-Header value")
	}
}

func TestRpcCallWithPositionalParamters(t *testing.T) {
	table := []struct {
		Body    string
		Jsonrpc string `json:"jsonrpc"`
		Result  int    `json:"result"`
		Id      int    `json:"id"`
	}{
		{`{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}`, "2.0", 19, 1},
		{`{"jsonrpc": "2.0", "method": "subtract", "params": [23, 42], "id": 2}`, "2.0", -19, 2},
	}

	for _, tc := range table {
		var result struct {
			Jsonrpc string      `json:"jsonrpc"`
			Result  int         `json:"result"`
			Error   interface{} `json:"error"`
			Id      int         `json:"id"`
		}

		buf := bytes.NewBuffer([]byte(tc.Body))
		resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		dec := json.NewDecoder(rdr)
		dec.Decode(&result)

		if result.Error != nil {
			t.Fatal("Expected error to be nil")
		}
		if result.Jsonrpc != tc.Jsonrpc {
			t.Fatal("Invalid jsonrpc member value")
		}
		if result.Result != tc.Result {
			t.Fatalf("Expected result to be %d", tc.Result)
		}
		if result.Id != tc.Id {
			t.Fatalf("Expected id to be %d", tc.Id)
		}
	}
}

func TestRpcCallWithPositionalParamtersError(t *testing.T) {
	var result struct {
		Jsonrpc string      `json:"jsonrpc"`
		Result  interface{} `json:"result"`
		Err     ErrorObject `json:"error"`
		Id      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": "subtract", "params": [999, 999], "id": 1}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)

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
	if result.Id != 1 {
		t.Fatal("Expected id to be 1")
	}
}

func TestRpcCallWithNamedParameters(t *testing.T) {
	table := []struct {
		Body    string
		Jsonrpc string `json:"jsonrpc"`
		Result  int    `json:"result"`
		Id      int    `json:"id"`
	}{
		{`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}`, "2.0", 19, 3},
		{`{"jsonrpc": "2.0", "method": "subtract", "params": {"minuend": 42, "subtrahend": 23}, "id": 4}`, "2.0", 19, 4},
	}

	for _, tc := range table {
		var result struct {
			Jsonrpc string      `json:"jsonrpc"`
			Result  int         `json:"result"`
			Error   interface{} `json:"error"`
			Id      int         `json:"id"`
		}

		buf := bytes.NewBuffer([]byte(tc.Body))
		resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		dec := json.NewDecoder(rdr)
		dec.Decode(&result)

		if result.Error != nil {
			t.Fatal("Expected error to be nil")
		}
		if result.Jsonrpc != tc.Jsonrpc {
			t.Fatal("Invalid jsonrpc member value")
		}
		if result.Result != tc.Result {
			t.Fatalf("Expected result to be %d", tc.Result)
		}
		if result.Id != tc.Id {
			t.Fatalf("Expected id to be %d", tc.Id)
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
		resp, err := http.Post("https://localhost:31511/api/v4/rpc", "application/json", buf)
		if err != nil {
			t.Fatal(err)
		}
		rdr := bufio.NewReader(resp.Body)
		data, err := rdr.ReadBytes('\b')

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
		Id      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": "fooba", "id": "1"}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)

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
		Id      int         `json:"id"`
	}
	body := `{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("https://localhost:31511/api/v4/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)

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
		Id      int         `json:"id"`
	}

	body := `{"jsonrpc": "2.0", "method": 1, "params": "bar"}`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)

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
		Id      int         `json:"id"`
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
	dec.Decode(&result)

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
		Id      int         `json:"id"`
	}

	body := `[]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)

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
		Id      int         `json:"id"`
	}

	body := `[]`
	buf := bytes.NewBuffer([]byte(body))
	resp, err := http.Post("http://localhost:31500/api/v1/rpc", "application/json", buf)
	if err != nil {
		t.Fatal(err)
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&results)

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
	if len(data) > 0 {
		t.Fatal("Expected batch notification to return no response body")
	}
}

func TestCallWithContext(t *testing.T) {
	body := `
        {"jsonrpc": "2.0", "method": "say", "params": ["Hello"], "id": 1}
	`
	req, err := http.NewRequest("POST", "http://localhost:31500/api/v1/rpc", bytes.NewBuffer([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("user", "bob")
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Jsonrpc string       `json:"jsonrpc"`
		Error   *ErrorObject `json:"error"`
		Result  interface{}  `json:"result"`
		Id      interface{}  `json:"id"`
	}
	rdr := bufio.NewReader(resp.Body)
	dec := json.NewDecoder(rdr)
	dec.Decode(&result)
	if result.Error != nil {
		fmt.Println(*result.Error)
		t.Fatal("Expected error to be nil")
	}
	if result.Result != "Hello bob!" {
		fmt.Println(result.Result)
		t.Fatal("Wrong result")
	}
}
