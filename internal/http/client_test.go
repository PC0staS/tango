package http

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Do(t *testing.T) {
	// Crear mock server
	server := httptest.NewServer(nil)
	defer server.Close()

	client := NewClient(5 * time.Second)
	req := &Request{
		Method: "GET",
		URL:    server.URL,
	}

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}