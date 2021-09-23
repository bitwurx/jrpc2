package jrpc2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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
MIIDBzCCAe+gAwIBAgIUVnylqJoVhV2TkZyMnoG5d1WntGcwDQYJKoZIhvcNAQEL
BQAwEzERMA8GA1UEAwwIdGVzdC5jb20wHhcNMjAwMTMwMTkwMTA0WhcNMjEwMTI5
MTkwMTA0WjATMREwDwYDVQQDDAh0ZXN0LmNvbTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAMGrFhH2esQ6kpRzZaWcB8Gv+P3oD+VGP6rnAGhItNdI//sg
DvPJY9Icl3KJCfLBNeFlZ2b9bnJqX1Q04+ZOPniJBvAEaOFmMRZShV/61BUpdH7o
BVMCPfcOvtj/u3Q/rSNOwtoipzkARwgOogak20MIy5s5z44NYHgALDjU7DNxsiep
VprFXEVvYFlZbwyAAn9FeK1q0WbSH7yNe0STgDgoB5o5C/NCWzT/lE+GXFnwRphC
OLCn3tpdHpO1TruUEDd6ejIOfbq4o2Rar1WA3X8wUkKKrRRWoKYWNsGz2NKlP+Dr
l/u+5s4z0tWUmwQ0IYEng4lehV0AM2AEQ33t5fsCAwEAAaNTMFEwHQYDVR0OBBYE
FD7rpYF1pvkAl2ZiUq7rl2Ek0+hXMB8GA1UdIwQYMBaAFD7rpYF1pvkAl2ZiUq7r
l2Ek0+hXMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAIPqxd88
Hq+crhH1HlJCphlLXgKDLRaqYYxmhr7c8BCmJC23KLilAEsSC/CCUMi+ZPgoLDhs
n2qTYTu0FNWR0OaHfH9JIuC5c+/L/ppFyiQOtU4t1uU12/xcVbKDeGt48zoyxkDG
gY662VrkNbqF5SA45OVhFoVwxeRnxoSuTTAM38Ai4PjvTEg5VLeV53j17Vxq/es6
UgGKCPWbWOxtbOY1ZgQqCXJnfn6GogGEs+l1Ww6IwmtnhtUQtGs96cGdKw9Vtbrp
GiDcGN2r88C+o32l3F4Gco9X5iVCQz4RkO1gOaj1IZ2136g3Ko3e/YGx3R7j+4ze
b3PnPxfPhlb8D6k=
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
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDBqxYR9nrEOpKU
c2WlnAfBr/j96A/lRj+q5wBoSLTXSP/7IA7zyWPSHJdyiQnywTXhZWdm/W5yal9U
NOPmTj54iQbwBGjhZjEWUoVf+tQVKXR+6AVTAj33Dr7Y/7t0P60jTsLaIqc5AEcI
DqIGpNtDCMubOc+ODWB4ACw41OwzcbInqVaaxVxFb2BZWW8MgAJ/RXitatFm0h+8
jXtEk4A4KAeaOQvzQls0/5RPhlxZ8EaYQjiwp97aXR6TtU67lBA3enoyDn26uKNk
Wq9VgN1/MFJCiq0UVqCmFjbBs9jSpT/g65f7vubOM9LVlJsENCGBJ4OJXoVdADNg
BEN97eX7AgMBAAECggEADsK+bOIPW1Nnhp8A+U1aHf4OiTOduojPI3R1yHz6I4px
0C8SVKxdyk7ZkCY3tuPY+nPjHKtmNpw65c0eLZh7FG7FM5fycnN6fEwP1E/myDIf
qeh/N2NtW54pF5ruK58K0C0ZlsybWDHYOBn9aWo5N/O8qPkQA7CrUJoaxL4dvpHi
qh6o/sZ/SA9YdjRB5mmF2U8Fx/A7blhtN62XskBttX4RZTW1szgZnN5QzcakCQUw
rCrzwEzvNT3WWJdcZ4skt7QMqGkfkE3u00UVKr/o5WVKoJSWWcVHmIR7ioKegRKa
CVJpCEE7dGI461A8degduZwjbL2VXOwBNWYT26TvIQKBgQD5677EQnpt7E/so4gj
lvZW2/XLNWN7FuR7vh9ecKy1sVC4AobtZdOIlI54vOsuO+OqqKQ13nLPGF8DZRSx
QFo5Wiz86y71Mbk54YSxVoJt27oc8Zn+86vzZM6xuB2/8a6KpGTP10xC6UXTnRYx
fexhLteGsfZaLrdu4qMh97/3KwKBgQDGYQ6LTRg9mvLWL9x2UnXSphXckh1oUrQB
XkeajhdG3cZNHkaziyMH4N5KJ8aZgZFQec9RofHYWJ4iO5vmd9C1qViKhv/s6hfn
Uo/bqpRC+6fRfDyJGxlpdcGzHNk8TyYB3GhSBoDnJ9RA2JhBIZfgNr8MHtnJkpBo
/FBo7m9kcQKBgHE3i8cq+n17lUV1W8ILrHLy2GmDORrU5xLrsRg+YO86cX+6nVdE
TszLx7MIml3qgZuZJDLHICmTN8+45ePabEUZBdJZ1H79VJTVBiC0OQf9h1V/Waz2
xEnRvBUkfE2s9c4W5RiGxyR0us4/loM7MW9hIgAB9MEr8qtH/nDv5EXbAoGBALlv
jlXeifNEPQzEDnO4HxT6VWMqXjzfWg4RYCN0AQQoWK5Lx9EbFXLO21s8FSP2/qvY
QVhQZi5Sn/bl+5QSmdDF7NMI4IBITnHYNksjB5YZgUSLulZ7M2TmQ1s3c0Uxwxho
PEe4dpQdIgY/sQro6PwYkLs2t2P6Ee1hNZTwlMWxAoGAGecr2LApI2beTma0kd5t
WzoVUIF/vTrILivPZT7/TbLgxCKPKIu6T4GnlflJFTL1VgLU47fXOWmAQhG/Vgmq
NzcjvvSNVQBUflIgNasVp6cSxtAoc8QVd9s5NQ6DUUT0xpdnfiGrIrWqej6soegC
xWP/0/cB4FI0YRji1toIZtY=
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
		resp, err := http.Post("http://localhost:31511/api/v4/rpc", "application/json", buf)
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
	resp, err := http.Post("http://localhost:31511/api/v4/rpc", "application/json", buf)
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
