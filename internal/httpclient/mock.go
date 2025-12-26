package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	requests  []*http.Request
}

// NewMockHTTPClient creates a new MockHTTPClient instance
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
		requests:  make([]*http.Request, 0),
	}
}

// SetResponse sets a mock response for a URL
func (m *MockHTTPClient) SetResponse(url string, statusCode int, body string, headers map[string]string) {
	bodyReader := io.NopCloser(bytes.NewReader([]byte(body)))
	resp := &http.Response{
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Body:       bodyReader,
		Header:     make(http.Header),
	}

	if headers != nil {
		for k, v := range headers {
			resp.Header.Set(k, v)
		}
	}

	m.responses[url] = resp
}

// SetError sets an error to return for a URL
func (m *MockHTTPClient) SetError(url string, err error) {
	m.errors[url] = err
}

// GetRequests returns all requests made to this client
func (m *MockHTTPClient) GetRequests() []*http.Request {
	return m.requests
}

// ClearRequests clears the request history
func (m *MockHTTPClient) ClearRequests() {
	m.requests = make([]*http.Request, 0)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Store request
	m.requests = append(m.requests, req)

	url := req.URL.String()

	// Check for error
	if err, ok := m.errors[url]; ok {
		return nil, err
	}

	// Check for response
	if resp, ok := m.responses[url]; ok {
		// Create a new response with a new body reader (body can only be read once)
		bodyBytes := make([]byte, 0)
		if resp.Body != nil {
			bodyBytes, _ = io.ReadAll(resp.Body)
		}
		return &http.Response{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
			Header:     resp.Header,
		}, nil
	}

	// Default: return 404
	return &http.Response{
		Status:     http.StatusText(http.StatusNotFound),
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte("Not Found"))),
		Header:     make(http.Header),
	}, nil
}





