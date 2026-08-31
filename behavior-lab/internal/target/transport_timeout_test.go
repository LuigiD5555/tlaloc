package target

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestHTTPClientUsesContextDeadlineWhenNoExplicitTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := (OpenAICompat{}).httpClient(ctx)
	if client.Timeout <= 2*time.Second || client.Timeout > 3*time.Second {
		t.Fatalf("context deadline not reflected in client timeout: %s", client.Timeout)
	}
}

func TestHTTPClientExplicitRequestTimeoutWins(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := (OpenAICompat{RequestTimeout: 7 * time.Second}).httpClient(ctx)
	if client.Timeout != 7*time.Second {
		t.Fatalf("explicit request timeout lost: %s", client.Timeout)
	}
}

func TestHTTPClientInjectionWins(t *testing.T) {
	injected := &http.Client{Timeout: 11 * time.Second}
	client := (OpenAICompat{Client: injected, RequestTimeout: 7 * time.Second}).httpClient(context.Background())
	if client != injected {
		t.Fatal("injected client was not preserved")
	}
}
