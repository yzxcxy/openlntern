package embedding

import (
	"context"
	"testing"
	"time"

	"openIntern/internal/config"
)

func TestEmbedStrings(t *testing.T) {
	model := "doubao-embedding-large-text-250515"
	apiKey := "9f1c2796-55d4-416e-ab08-46c711294934"
	if model == "" || apiKey == "" {
		t.Fatalf("EMBEDDING_MODEL or EMBEDDING_API_KEY not set")
	}

	if err := InitEmbedding(config.EmbeddingLLMConfig{Model: model, APIKey: apiKey}); err != nil {
		t.Fatalf("init embedding failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	vectors, err := Embedding.EmbedStrings(ctx, []string{"hello"})
	if err != nil {
		t.Fatalf("embed strings failed: %v", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		t.Fatalf("empty embedding result")
	}
	t.Logf("dim: %d", len(vectors[0]))	
}
