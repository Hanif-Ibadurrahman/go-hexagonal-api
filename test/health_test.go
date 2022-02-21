package test

import (
	"fmt"
	"net/http"
	"testing"
)

func Test_HealthCheck_Ok(t *testing.T) {
	// Arrange
	cfg := New(t)

	// Act
	url := fmt.Sprintf("%s:%d/api/health", cfg.Address, cfg.Port)

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	// Assert
	if want, got := http.StatusOK, resp.StatusCode; want != got {
		t.Fatalf("unexpected http status code while calling %s: want=%d but got=%d", resp.Request.URL, want, got)
	}
}
