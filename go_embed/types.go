package goembed

type Chunk struct {
	ID   string `json:"id"`
	File string `json:"file"`
	Code string `json:"code"`
}

type EmbeddedChunk struct {
	ID        string
	File      string
	Code      string
	Embedding []float64
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type ScoredChunk struct {
	ID    string
	File  string
	Code  string
	Score float64
}
