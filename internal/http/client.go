package http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

type Request struct {
	Method string
	URL string
	Headers map[string]string
	Body string
}

type Response struct {
	StatusCode int
	Headers map[string]string
	Body string
	Duration time.Duration
}

type Client struct{
	httpClient *http.Client
	timeout time.Duration
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		timeout: timeout,
	}
}

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	// 1. Create http.Request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(req.Body))
	if err != nil {
		return nil, err
	}

	// 2. Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	// 3.Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// 4. Read body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	
	// 6. Convert headers
	respHeaders := make(map[string]string)
	for k, v := range httpResp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	
	return &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    respHeaders,
		Body:       string(bodyBytes),
		Duration:   time.Since(start),
	}, nil
}