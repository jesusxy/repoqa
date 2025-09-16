package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	goembed "repoqa/go_embed"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "repoqa",
		Usage: "Semantic code search via OpenAI embeddings",
		Commands: []*cli.Command{
			{
				Name:  "embed",
				Usage: "Read chunked.jsonl, embed code and write .index.json",
				Action: func(c *cli.Context) error {
					return handleEmbed()
				},
			},
			{
				Name:  "query",
				Usage: "",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "top",
						Aliases: []string{"t"},
						Value:   3,
						Usage:   "Number of top matching chunks to return",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output results as raw JSON for parsing.",
					},
				},
				Action: func(c *cli.Context) error {
					query := c.Args().First()
					if query == "" {
						return cli.Exit("Error: query string required.\nUsage: repoqa query \"your question\" --top 5", 1)
					}
					return handleQuery(query, c.Int("top"), c.Bool("json"))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func handleEmbed() error {
	start := time.Now()
	var chunks []goembed.Chunk

	// open and read file line by line
	file, err := os.Open("data/chunked.jsonl")
	if err != nil {
		log.Fatalf("failed to open chunked.jsonl: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		var chunk goembed.Chunk
		err := json.Unmarshal([]byte(line), &chunk)

		if err != nil {
			fmt.Printf("Skipping invalid line: %v\n", err)
			continue
		}

		if chunk.File == "" || chunk.Code == "" {
			fmt.Printf("Skipping incomplete chunk: %+v\n", chunk)
			continue
		}

		chunks = append(chunks, chunk)
	}
	fmt.Printf("Parsed %d chunks\n", len(chunks))

	embedded := goembed.EmbedChunks(chunks)

	if err := goembed.WriteIndex(embedded, "data/.index.json"); err != nil {
		log.Fatalf("failed to write index: %v", err)
	}
	fmt.Println("✅ Successfully wrote embeddings to .index.json")
	log.Printf("⏱ Completed in %s\n", time.Since(start).Round(time.Millisecond))
	rate := float64(len(embedded)) / time.Since(start).Seconds()
	log.Printf("⚡️ %.2f chunks/sec", rate)
	return nil
}

func handleQuery(query string, topK int, asJSON bool) error {
	start := time.Now()

	results, err := goembed.SearchIndex(query, "data/.index.json", topK)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	if asJSON {
		// Clean machine-readable output
		jsonBytes, err := json.Marshal(results)
		if err != nil {
			return fmt.Errorf("failed to encode results: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	for i, match := range results {
		fmt.Printf("### Match #%d — Score: %.4f\n", i+1, match.Score)
		fmt.Printf("📄 File: %s\n\n", match.File)
		fmt.Println("```go")
		fmt.Println(match.Code)
		fmt.Println("```")
		fmt.Println()
	}

	log.Printf("🔎 Query completed in %s", time.Since(start).Round(time.Millisecond))
	return nil
}
