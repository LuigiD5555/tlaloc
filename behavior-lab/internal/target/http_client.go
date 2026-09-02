package target

import (
	"context"
	"net/http"
	"time"
)

// httpClientForTimeout makes the request context the timeout authority
// unless a caller explicitly injects a client or timeout. This prevents a
// hidden fixed transport timeout from cutting off longer campaign calls
// early. Shared by OpenAICompat and Embeddings.
func httpClientForTimeout(ctx context.Context, client *http.Client, timeout time.Duration) *http.Client {
	if client != nil {
		return client
	}
	if timeout <= 0 {
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
			if timeout <= 0 {
				timeout = time.Millisecond
			}
		} else {
			timeout = 90 * time.Second
		}
	}
	return &http.Client{Timeout: timeout}
}
