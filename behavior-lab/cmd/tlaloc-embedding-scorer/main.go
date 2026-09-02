// Command tlaloc-embedding-scorer is a resident micro-model Tlaloque: it
// keeps talking to a small, always-loaded embedding model (e.g. LM Studio's
// text-embedding-all-minilm-l6-v2-embedding) and scores answer relevance by
// cosine similarity instead of a full chat-completion judge. It speaks the
// tlaloque.CapabilityRequest/CapabilityResponse HTTP_JSON contract that
// internal/tlaloque/http_worker.go already knows how to consume.
package main

import (
	"flag"
	"log"
	"net/http"

	"tlaloc.local/behaviorlab/internal/target"
)

func main() {
	addr := flag.String("addr", ":8790", "address to listen on")
	lmStudioURL := flag.String("lmstudio-url", "http://127.0.0.1:1234/v1", "LM Studio API base URL")
	embeddingModel := flag.String("embedding-model", "text-embedding-all-minilm-l6-v2-embedding", "resident embedding model id")
	flag.Parse()

	embedder := target.Embeddings{BaseURL: *lmStudioURL, Model: *embeddingModel}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", scoreHandler(embedder))
	mux.HandleFunc("/health", healthHandler(*embeddingModel))

	log.Printf("tlaloc-embedding-scorer listening on %s (embedding model %q via %s)", *addr, *embeddingModel, *lmStudioURL)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
