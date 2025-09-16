package goembed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func EmbedChunks(chunks []Chunk) []EmbeddedChunk {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	results := make(chan EmbeddedChunk, len(chunks))

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	for _, chunk := range chunks {
		wg.Add(1)

		go func(c Chunk) {
			defer wg.Done()

			req, err := buildRequest(c, apiKey)
			if err != nil {
				log.Printf("Failed to build request for %s: %v", c.ID, err)
				return
			}

			sem <- struct{}{}
			defer func() { <-sem }()

			var resp *http.Response
			maxRetries := 3

			for attempt := 1; attempt <= maxRetries; attempt++ {
				resp, err = httpClient.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					break
				}

				// Read body if response exists (to avoid leaking connections)
				if resp != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}

				wait := time.Duration(attempt*500) * time.Millisecond
				log.Printf("Retry %d for chunk %s: %v (status: %d)", attempt, c.ID, err, resp.StatusCode)
				time.Sleep(wait)
			}

			if err != nil {
				log.Printf("Final failure for chunk %s: %v", c.ID, err)
				return
			}

			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body for %s: %v", c.ID, err)
				return
			}

			var result EmbeddingResponse
			if err := json.Unmarshal(body, &result); err != nil {
				log.Printf("Unmarshal error for %s: %v", c.ID, err)
				return
			}

			if len(result.Data) == 0 {
				log.Printf("No embedding returned for chunk %s", c.ID)
				return
			}

			log.Printf("Successfully embedded chunk: %s", c.ID)

			results <- EmbeddedChunk{
				ID:        c.ID,
				File:      c.File,
				Code:      c.Code,
				Embedding: result.Data[0].Embedding,
			}

		}(chunk)
	}

	wg.Wait()
	close(results)

	var embedded []EmbeddedChunk
	for ec := range results {
		if isValidEmbedding(ec) {
			embedded = append(embedded, ec)
		} else {
			log.Printf("Invalid embedding for chunk %s — skipped", ec.ID)
		}
	}

	log.Printf("Validated and retained %d of %d embeddings", len(embedded), len(chunks))

	return embedded

}

func WriteIndex(chunks []EmbeddedChunk, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to open path %s: %w", path, err)
	}

	defer f.Close()

	serializedChunks, err := json.MarshalIndent(chunks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshall chunks: %w", err)
	}

	_, err = f.Write(serializedChunks)
	if err != nil {
		return fmt.Errorf("failed to marshal chunks: %w", err)

	}

	log.Printf("Wrote %d embedded chunks to %s", len(chunks), path)

	return nil
}

func EmbedQuery(input string, apiKey string) ([]float64, error) {
	const endpoint = "https://api.openai.com/v1/embeddings"
	postBody, err := json.Marshal(map[string]string{
		"input": input,
		"model": "text-embedding-ada-002",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(body[:200]))
	}

	defer resp.Body.Close()

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned for query input")
	}

	return result.Data[0].Embedding, nil
}

func buildRequest(chunk Chunk, apiKey string) (*http.Request, error) {
	ENDPOINT := "https://api.openai.com/v1/embeddings"
	postBody, _ := json.Marshal(map[string]string{
		"input": chunk.Code,
		"model": "text-embedding-ada-002",
	})

	req, err := http.NewRequest("POST", ENDPOINT, bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func isValidEmbedding(e EmbeddedChunk) bool {
	if len(e.Embedding) != 1536 {
		return false
	}

	for _, v := range e.Embedding {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}

	return true
}
