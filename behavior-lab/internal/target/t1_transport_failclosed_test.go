package target

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TONAL T1 transport-repair requirement: an HTTP 200 is NOT sufficient for
// success. These tests exercise the exact seven cases the pre-inference
// mandate requires before any live T1 call: valid text, valid multimodal,
// HTTP error, HTTP 200 with an embedded {"error":...} body, malformed JSON,
// empty choices, and empty content. Every failure case must return a
// non-nil error — raw_output=="" must never be silently accepted as success.

func TestT1Transport_ValidTextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"420"}}],"usage":{"prompt_tokens":5,"completion_tokens":1}}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	result, err := client.CompleteText(context.Background(), "", "Which is larger?")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Content != "420" {
		t.Fatalf("content = %q, want 420", result.Content)
	}
}

func TestT1Transport_ValidMultimodalResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"95"}}],"usage":{"prompt_tokens":40,"completion_tokens":1}}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	result, err := client.CompletePerception(context.Background(), PerceptionInput{
		Question: "Read the number.", Image: []byte("fake-png-bytes"), MediaType: "image/png",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Content != "95" {
		t.Fatalf("content = %q, want 95", result.Content)
	}
}

func TestT1Transport_HTTPErrorFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `internal server error`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	_, err := client.CompleteText(context.Background(), "", "test")
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}

	_, err = client.CompletePerception(context.Background(), PerceptionInput{
		Question: "test", Image: []byte("x"), MediaType: "image/png",
	})
	if err == nil {
		t.Fatal("expected error on HTTP 500 (perception), got nil")
	}
}

// TestT1Transport_HTTP200WithErrorBodyFailsClosed reproduces the EXACT
// failure mode from TONAL T1 attempt 0: LM Studio returns HTTP 200 with a
// JSON body of {"error": "..."} for an unrecognized endpoint/payload shape.
// The real target.OpenAICompat client must reject this (zero Choices),
// unlike the ad-hoc script that silently treated it as empty success.
func TestT1Transport_HTTP200WithErrorBodyFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"error":"Unexpected endpoint or method."}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	result, err := client.CompleteText(context.Background(), "", "test")
	if err == nil {
		t.Fatalf("expected error for HTTP 200 error-body, got success with content=%q", result.Content)
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("expected 'no choices' error, got: %v", err)
	}

	presult, perr := client.CompletePerception(context.Background(), PerceptionInput{
		Question: "test", Image: []byte("x"), MediaType: "image/png",
	})
	if perr == nil {
		t.Fatalf("expected error for HTTP 200 error-body (perception), got success with content=%q", presult.Content)
	}
}

func TestT1Transport_MalformedJSONFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{not valid json`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	_, err := client.CompleteText(context.Background(), "", "test")
	if err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}

	_, err = client.CompletePerception(context.Background(), PerceptionInput{
		Question: "test", Image: []byte("x"), MediaType: "image/png",
	})
	if err == nil {
		t.Fatal("expected error on malformed JSON (perception), got nil")
	}
}

func TestT1Transport_EmptyChoicesFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[],"usage":{"prompt_tokens":5,"completion_tokens":0}}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	_, err := client.CompleteText(context.Background(), "", "test")
	if err == nil {
		t.Fatal("expected error on empty choices, got nil")
	}

	_, err = client.CompletePerception(context.Background(), PerceptionInput{
		Question: "test", Image: []byte("x"), MediaType: "image/png",
	})
	if err == nil {
		t.Fatal("expected error on empty choices (perception), got nil")
	}
}

// TestT1Transport_EmptyContentFailsClosed: choices present but content is
// empty/whitespace. This is the exact silent-success bug from attempt 0 —
// raw_output=="" must never be treated as a valid completion.
func TestT1Transport_EmptyContentFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"   "}}],"usage":{"prompt_tokens":5,"completion_tokens":0}}`)
	}))
	defer server.Close()

	client := OpenAICompat{BaseURL: server.URL + "/v1", Model: "lfm2-vl-1.6b"}
	_, err := client.CompletePerception(context.Background(), PerceptionInput{
		Question: "test", Image: []byte("x"), MediaType: "image/png",
	})
	if err == nil {
		t.Fatal("expected error on empty/whitespace content, got nil — this is the exact attempt-0 bug")
	}
	if !strings.Contains(err.Error(), "empty perception content") {
		t.Fatalf("expected 'empty perception content' error, got: %v", err)
	}
}

// TestT1Transport_ConnectionRefusedFailsClosed simulates an unreachable
// endpoint (e.g. LM Studio not running).
func TestT1Transport_ConnectionRefusedFailsClosed(t *testing.T) {
	client := OpenAICompat{BaseURL: "http://127.0.0.1:1", Model: "lfm2-vl-1.6b"} // port 1: nothing listens
	_, err := client.CompleteText(context.Background(), "", "test")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}
