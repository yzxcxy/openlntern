package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"openIntern/internal/config"
	"openIntern/internal/services/embedding"
	"strings"

	milvus2 "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	einoembedding "github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

type Service struct {
	cfg config.MilvusConfig
}

var RAG = new(Service)

type ctxKey string

const userIDContextKey ctxKey = "user_id"

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func (s *Service) Indexer(collectionName string) indexer.Indexer {
	return &ragIndexer{
		service:        s,
		collectionName: collectionName,
	}
}

func (s *Service) Retriever(collectionName string) retriever.Retriever {
	return &ragRetriever{
		service:        s,
		collectionName: collectionName,
	}
}

func InitRAG(cfg config.MilvusConfig) error {
	if cfg.Address == "" {
		return errors.New("milvus address required")
	}
	if cfg.Dimension <= 0 {
		return errors.New("milvus dimension required")
	}
	if cfg.MetricType == "" {
		return errors.New("milvus metric type required")
	}
	if cfg.EnableHybrid && strings.TrimSpace(cfg.AnalyzerType) == "" {
		return errors.New("milvus analyzer_type required when hybrid enabled")
	}
	RAG.cfg = cfg
	return nil
}

// EnsureUserDatabase 确保用户数据库存在(一个用户一个数据库)
func (s *Service) EnsureUserDatabase(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("user_id required")
	}
	dbName := s.DatabaseName(userID)
	client, err := s.newClient(ctx, dbName)
	if err != nil {
		return err
	}
	defer client.Close(ctx)
	_, err = client.DescribeDatabase(ctx, milvusclient.NewDescribeDatabaseOption(dbName))
	if err != nil {
		if err := client.CreateDatabase(ctx, milvusclient.NewCreateDatabaseOption(dbName)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) EnsureMemoryCollection(ctx context.Context, userID string) error {
	return s.ensureCollection(ctx, userID, "memory")
}

func (s *Service) CreateKnowledgeBaseCollection(ctx context.Context, userID string, kbID string) error {
	if kbID == "" {
		return errors.New("kb_id required")
	}
	return s.ensureCollection(ctx, userID, s.KBCollectionName(kbID))
}

func (s *Service) DatabaseName(userID string) string {
	return "user_" + userID
}

func (s *Service) KBCollectionName(kbID string) string {
	kbID = strings.TrimSpace(kbID)
	return "kb_" + kbID
}

func (s *Service) ensureCollection(ctx context.Context, userID string, collectionName string) error {
	if userID == "" {
		return errors.New("user_id required")
	}
	client, idx, err := s.newIndexer(ctx, userID, collectionName, nil)
	if err != nil {
		return err
	}
	defer client.Close(ctx)
	_ = idx
	return nil
}

func (s *Service) newClient(ctx context.Context, dbName string) (*milvusclient.Client, error) {
	cfg := &milvusclient.ClientConfig{
		Address:  s.cfg.Address,
		Username: s.cfg.Username,
		Password: s.cfg.Password,
		APIKey:   s.cfg.APIKey,
	}
	if dbName != "" {
		cfg.DBName = dbName
	}
	return milvusclient.New(ctx, cfg)
}

func (s *Service) newIndexer(ctx context.Context, userID string, collectionName string, emb einoembedding.Embedder) (*milvusclient.Client, *milvus2.Indexer, error) {
	client, err := s.newClient(ctx, s.DatabaseName(userID))
	if err != nil {
		return nil, nil, err
	}
	indexerCfg := &milvus2.IndexerConfig{
		Client:     client,
		Collection: collectionName,
		Vector: &milvus2.VectorConfig{
			Dimension:  s.cfg.Dimension,
			MetricType: milvus2.MetricType(s.cfg.MetricType),
		},
		Embedding: emb,
	}
	if s.cfg.EnableHybrid {
		indexerCfg.FieldParams = map[string]map[string]string{
			"content": {
				"enable_analyzer": "true",
				"analyzer_params": fmt.Sprintf(`{"type":"%s"}`, strings.TrimSpace(s.cfg.AnalyzerType)),
			},
		}
		indexerCfg.Sparse = &milvus2.SparseVectorConfig{
			VectorField: "sparse_vector",
			MetricType:  milvus2.BM25,
			Method:      milvus2.SparseMethodAuto,
		}
	}
	idx, err := milvus2.NewIndexer(ctx, indexerCfg)
	if err != nil {
		_ = client.Close(ctx)
		return nil, nil, err
	}
	return client, idx, nil
}

func (s *Service) embedder() einoembedding.Embedder {
	return embedding.Embedding.AsEmbedder()
}

type ragIndexer struct {
	service        *Service
	collectionName string
}

func (r *ragIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	emb := r.service.embedder()
	if emb == nil {
		return nil, errors.New("embedding not configured")
	}
	indexerClient, idx, err := r.service.newIndexer(ctx, userID, r.collectionName, emb)
	if err != nil {
		return nil, err
	}
	defer indexerClient.Close(ctx)
	co := indexer.GetCommonOptions(&indexer.Options{}, opts...)
	if co.Embedding == nil {
		opts = append(opts, indexer.WithEmbedding(emb))
	}
	return idx.Store(ctx, docs, opts...)
}

type ragRetriever struct {
	service        *Service
	collectionName string
}

func (r *ragRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	co := retriever.GetCommonOptions(&retriever.Options{}, opts...)
	topK := 5
	if co.TopK != nil {
		topK = *co.TopK
	}
	collectionName := r.collectionName
	if co.Index != nil && *co.Index != "" {
		collectionName = *co.Index
	} else if co.SubIndex != nil && *co.SubIndex != "" {
		collectionName = *co.SubIndex
	}
	emb := co.Embedding
	if emb == nil {
		emb = r.service.embedder()
	}
	if emb == nil {
		return nil, errors.New("embedding not configured")
	}
	vectors, err := emb.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("empty embedding result")
	}
	vector := toFloat32Vector(vectors[0])
	client, err := r.service.newClient(ctx, r.service.DatabaseName(userID))
	if err != nil {
		return nil, err
	}
	defer client.Close(ctx)
	loadTask, err := client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collectionName))
	if err != nil {
		return nil, err
	}
	if err := loadTask.Await(ctx); err != nil {
		return nil, err
	}
	if r.service.cfg.EnableHybrid {
		denseReq := milvusclient.NewAnnRequest("vector", topK, entity.FloatVector(vector)).
			WithSearchParam("metric_type", r.service.cfg.MetricType)
		sparseReq := milvusclient.NewAnnRequest("sparse_vector", topK, entity.Text(query)).
			WithSearchParam("metric_type", "BM25")
		hybridOpt := milvusclient.NewHybridSearchOption(
			collectionName,
			topK,
			denseReq,
			sparseReq,
		).WithOutputFields("id", "content", "metadata").WithReranker(milvusclient.NewRRFReranker())
		resultSets, err := client.HybridSearch(ctx, hybridOpt)
		if err != nil {
			return nil, err
		}
		if len(resultSets) == 0 {
			return []*schema.Document{}, nil
		}
		return buildDocuments(resultSets[0], "HYBRID", nil)
	}
	searchOpt := milvusclient.NewSearchOption(
		collectionName,
		topK,
		[]entity.Vector{entity.FloatVector(vector)},
	).WithANNSField("vector").WithOutputFields("id", "content", "metadata").WithSearchParam("metric_type", r.service.cfg.MetricType)
	resultSets, err := client.Search(ctx, searchOpt)
	if err != nil {
		return nil, err
	}
	if len(resultSets) == 0 {
		return []*schema.Document{}, nil
	}
	return buildDocuments(resultSets[0], r.service.cfg.MetricType, co.ScoreThreshold)
}

func buildDocuments(rs milvusclient.ResultSet, metricType string, scoreThreshold *float64) ([]*schema.Document, error) {
	count := rs.ResultCount
	if count <= 0 {
		count = rs.IDs.Len()
	}
	if count <= 0 {
		return []*schema.Document{}, nil
	}
	idCol := rs.GetColumn("id")
	if idCol == nil {
		idCol = rs.IDs
	}
	contentCol := rs.GetColumn("content")
	metaCol := rs.GetColumn("metadata")
	docs := make([]*schema.Document, 0, count)
	for i := 0; i < count; i++ {
		id, _ := idCol.GetAsString(i)
		content := ""
		if contentCol != nil {
			content, _ = contentCol.GetAsString(i)
		}
		meta, err := readMetadata(metaCol, i)
		if err != nil {
			return nil, err
		}
		doc := &schema.Document{
			ID:       id,
			Content:  content,
			MetaData: meta,
		}
		if i < len(rs.Scores) {
			score := float64(rs.Scores[i])
			if scoreThreshold != nil && !matchScore(metricType, score, *scoreThreshold) {
				continue
			}
			doc.WithScore(score)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func matchScore(metricType string, score float64, threshold float64) bool {
	switch strings.ToUpper(metricType) {
	case "IP", "COSINE":
		return score >= threshold
	case "L2":
		return score <= threshold
	default:
		return score >= threshold
	}
}

func readMetadata(col column.Column, idx int) (map[string]any, error) {
	if col == nil {
		return nil, nil
	}
	value, err := col.Get(idx)
	if err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case map[string]any:
		return v, nil
	case []byte:
		if len(v) == 0 {
			return nil, nil
		}
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, err
		}
		return m, nil
	case string:
		if v == "" {
			return nil, nil
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
}

func toFloat32Vector(vector []float64) []float32 {
	if len(vector) == 0 {
		return nil
	}
	result := make([]float32, 0, len(vector))
	for _, v := range vector {
		result = append(result, float32(v))
	}
	return result
}

func userIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("context is nil")
	}
	if value := ctx.Value(userIDContextKey); value != nil {
		return normalizeUserID(value)
	}
	if value := ctx.Value("user_id"); value != nil {
		return normalizeUserID(value)
	}
	return "", errors.New("user_id not found in context")
}

func normalizeUserID(value any) (string, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "", errors.New("user_id is empty")
		}
		return v, nil
	case fmt.Stringer:
		userID := v.String()
		if userID == "" {
			return "", errors.New("user_id is empty")
		}
		return userID, nil
	default:
		userID := fmt.Sprint(value)
		if userID == "" {
			return "", errors.New("user_id is empty")
		}
		return userID, nil
	}
}

func (s *Service) Embedder() einoembedding.Embedder {
	return embedding.Embedding.AsEmbedder()
}
