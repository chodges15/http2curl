package http2curl

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestGetCurlCommand(t *testing.T) {
	tests := []struct {
		name        string
		setupReq    func() *http.Request
		opts        []CurlOption
		wantCommand string
		wantErr     bool
	}{
		{
			name: "GZIP decompression with auto-decompress",
			setupReq: func() *http.Request {
				body := compressData([]byte(`{"test":"gzip"}`))
				req, _ := http.NewRequest("POST", "http://example.com", bytes.NewReader(body))
				req.Header.Set("Content-Encoding", "gzip")
				return req
			},
			opts: []CurlOption{WithAutoDecompressGZIP()},
			wantCommand: `curl -X 'POST' -d '{"test":"gzip"}' ` +
				`'http://example.com'`,
		},
		{
			name: "Invalid GZIP data with auto-decompress",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://example.com", bytes.NewReader([]byte{0x1, 0x2}))
				req.Header.Set("Content-Encoding", "gzip")
				return req
			},
			opts:    []CurlOption{WithAutoDecompressGZIP()},
			wantErr: true,
		},
		{
			name: "Compression flag enabled",
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "http://example.com", nil)
			},
			opts:        []CurlOption{WithCompression()},
			wantCommand: "curl -X 'GET' 'http://example.com' --compressed",
		},
		{
			name: "Multiple security options",
			setupReq: func() *http.Request {
				req := httptest.NewRequest("PUT", "https://example.com", strings.NewReader("data"))
				req.TLS = &tls.ConnectionState{}
				return req
			},
			opts: []CurlOption{WithInsecureSkipVerify()},
			wantCommand: `curl -k -X 'PUT' -d 'data' ` +
				`'https://example.com'`,
		},
		{
			name: "form data POST request",
			setupReq: func() *http.Request {
				form := url.Values{}
				form.Add("age", "10")
				form.Add("name", "Hudson")
				body := form.Encode()
				req, _ := http.NewRequest(http.MethodPost, "http://foo.com/cats", bytes.NewBufferString(body))
				req.Header.Set("API_KEY", "123")
				return req
			},
			wantCommand: `curl -X 'POST' -d 'age=10&name=Hudson' -H 'Api_key: 123' 'http://foo.com/cats'`,
		},
		{
			name: "JSON body PUT request",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", bytes.NewBufferString(`{"hello":"world","answer":42}`))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'PUT' -d '{"hello":"world","answer":42}' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "no body request",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", nil)
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'PUT' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "empty string body",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", bytes.NewBufferString(""))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'PUT' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "newline in body with escaped newlines",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", bytes.NewBufferString("hello\nworld"))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			opts:        []CurlOption{WithEscapedNewlines()},
			wantCommand: `echo -e 'hello\nworld' | curl -X 'POST' -d @- -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "newline in body without escaped newlines",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", bytes.NewBufferString("hello\nworld"))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'POST' -d 'hello\nworld' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "special characters in body",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("POST", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", bytes.NewBufferString(`Hello $123 o'neill -"-`))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'POST' -d 'Hello $123 o'\''neill -"-' -H 'Content-Type: application/json' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "additional headers",
			setupReq: func() *http.Request {
				payload := bytes.NewBufferString(`{"hello":"world","answer":42}`)
				req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", payload)
				req.Header.Set("X-Auth-Token", "private-token")
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCommand: `curl -X 'PUT' -d '{"hello":"world","answer":42}' -H 'Content-Type: application/json' -H 'X-Auth-Token: private-token' 'http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "HTTPS with insecure skip verify",
			setupReq: func() *http.Request {
				payload := bytes.NewBufferString(`{"hello":"world","answer":42}`)
				req, _ := http.NewRequest("PUT", "https://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", payload)
				req.Header.Set("X-Auth-Token", "private-token")
				req.Header.Set("Content-Type", "application/json")
				req.TLS = &tls.ConnectionState{}
				return req
			},
			opts:        []CurlOption{WithInsecureSkipVerify()},
			wantCommand: `curl -k -X 'PUT' -d '{"hello":"world","answer":42}' -H 'Content-Type: application/json' -H 'X-Auth-Token: private-token' 'https://www.example.com/abc/def.ghi?jlk=mno&pqr=stu'`,
		},
		{
			name: "server side request headers",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com/", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				req.Header.Set("User-Agent", "Go-http-client/1.1")
				return req
			},
			wantCommand: `curl -X 'GET' -H 'Accept-Encoding: gzip' -H 'User-Agent: Go-http-client/1.1' 'http://example.com/'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			command, err := GetCurlCommand(req, tt.opts...)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetCurlCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && command.String() != tt.wantCommand {
				t.Errorf("Got:\n%s\nWant:\n%s", command.String(), tt.wantCommand)
			}
		})
	}
}

func TestConcurrentCommandGeneration(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "http://example.com", nil)
			_, _ = GetCurlCommand(req)
		}()
	}
	wg.Wait()
}

func compressData(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	return buf.Bytes()
}
