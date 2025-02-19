package http2curl

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// CurlCommand holds configuration options for curl command generation
type CurlCommand struct {
	Command            []string
	InsecureSkipVerify bool // -k
	EnableCompression  bool // --compressed
	AutoDecompressGZIP bool // Automatically decompress GZIP request
}

// append appends a string to the CurlCommand
func (c *CurlCommand) append(newSlice ...string) {
	c.Command = append(c.Command, newSlice...)
}

// String returns a ready to copy/paste command
func (c *CurlCommand) String() string {
	return strings.Join(c.Command, " ")
}

// CurlOption defines the functional option type
type CurlOption func(command *CurlCommand)

// WithInsecureSkipVerify enables insecure SSL verification
func WithInsecureSkipVerify() CurlOption {
	return func(c *CurlCommand) {
		c.InsecureSkipVerify = true
	}
}

// WithCompression enables --compressed flag
func WithCompression() CurlOption {
	return func(c *CurlCommand) {
		c.EnableCompression = true
	}
}

// WithAutoDecompressGZIP enables automatic GZIP decompression
func WithAutoDecompressGZIP() CurlOption {
	return func(c *CurlCommand) {
		c.AutoDecompressGZIP = true
	}
}

// GetCurlCommand generates curl command with configurable options
func GetCurlCommand(req *http.Request, opts ...CurlOption) (*CurlCommand, error) {
	command := &CurlCommand{}
	command.append("curl")

	decompressedBody := false

	// Apply options
	for _, opt := range opts {
		opt(command)
	}

	// Configure SSL verification
	if command.InsecureSkipVerify && req.URL.Scheme == "https" {
		command.append("-k")
	}

	command.append("-X", bashEscape(req.Method))

	// Process request body
	if req.Body != nil {
		var buff bytes.Buffer
		if _, err := buff.ReadFrom(req.Body); err != nil {
			return nil, fmt.Errorf("buffer read error: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(buff.Bytes()))

		// Handle GZIP decompression if enabled
		if command.AutoDecompressGZIP && req.Header.Get("Content-Encoding") == "gzip" {
			decompressed, err := decompressGZIP(buff.Bytes())
			if err != nil {
				return nil, err
			}
			buff.Reset()
			buff.Write(decompressed)
			decompressedBody = true
		}

		if buff.Len() > 0 {
			command.append("-d", bashEscape(buff.String()))
		}
	}

	// Add headers
	for _, k := range sortedKeys(req.Header) {
		if decompressedBody && (k == "Content-Encoding" || k == "Content-Length") {
			continue
		}
		command.append("-H", bashEscape(fmt.Sprintf("%s: %s", k, strings.Join(req.Header[k], " "))))
	}

	command.append(bashEscape(requestURL(req)))

	if command.EnableCompression {
		command.append("--compressed")
	}

	return command, nil
}

// Helper functions
func bashEscape(str string) string {
	return `'` + strings.Replace(str, `'`, `'\''`, -1) + `'`
}

func decompressGZIP(data []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip decompression failed: %w", err)
	}
	defer func(gzReader *gzip.Reader) {
		err := gzReader.Close()
		if err != nil {

		}
	}(gzReader)
	return io.ReadAll(gzReader)
}

func sortedKeys(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func requestURL(req *http.Request) string {
	if req.URL.Scheme == "" {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.Path)
	}
	return req.URL.String()
}
