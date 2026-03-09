// 保留，但是不使用
package embedding

import (
	"context"
	"errors"
	"openIntern/internal/config"

	"github.com/cloudwego/eino/components/embedding"
	arkembed "github.com/cloudwego/eino-ext/components/embedding/ark"
)

type EmbeddingService struct {
	embedder embedding.Embedder
}

var Embedding = new(EmbeddingService)

func InitEmbedding(cfg config.EmbeddingLLMConfig) error {
	if cfg.Model == "" || cfg.APIKey == "" {
		return errors.New("embedding llm config incomplete")
	}
	embedder, err := arkembed.NewEmbedder(context.Background(), &arkembed.EmbeddingConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	})
	if err != nil {
		return err
	}
	Embedding.embedder = embedder
	return nil
}

func (s *EmbeddingService) EmbedStrings(ctx context.Context, texts []string) ([][]float64, error) {
	if s.embedder == nil {
		return nil, errors.New("embedding service not configured")
	}
	return s.embedder.EmbedStrings(ctx, texts)
}

func (s *EmbeddingService) AsEmbedder() embedding.Embedder {
	return s.embedder
}
