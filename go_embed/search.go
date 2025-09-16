package goembed

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"sort"
)

func SearchIndex(query string, indexPath string, topK int) ([]ScoredChunk, error) {
	var matches []ScoredChunk
	var index []EmbeddedChunk

	f, err := os.Open(indexPath)
	if err != nil {
		log.Printf("Error opening file %s", indexPath)
		return nil, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&index); err != nil {
		return nil, err
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	queryEmbedding, err := EmbedQuery(query, apiKey)
	if err != nil {
		log.Printf("Error embedding query: %v", err)
		return nil, err
	}

	for _, chunk := range index {
		score := cosineSimilarity(queryEmbedding, chunk.Embedding)
		matches = append(matches, ScoredChunk{
			ID:    chunk.ID,
			File:  chunk.File,
			Code:  chunk.Code,
			Score: score,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) > topK {
		matches = matches[:topK]
	}

	log.Printf("Scored %d chunks, returning top %d", len(index), len(matches))

	return matches, nil
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
