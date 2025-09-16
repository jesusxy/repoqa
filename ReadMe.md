## repoqa

#### 🧪 Workflow

```
# Index a repo (only once or after changes)
$ repoqa index ./typescript-go

✅ Indexed 127 files, 1,600 code chunks.

# Ask questions about the repo
$ repoqa ask "How is type-checking handled in the parser?"

🧠 Searching...
🧠 Selecting top 3 chunks (3.2K tokens)
🧠 Asking GPT-4...

📎 Answer:
The parser calls into `checker.go`, which recursively validates types using...
```

#### Repo Scaffold

```
repoqa/
├── go_embed/
│   ├── main.go          # CLI entrypoint
│   ├── embed.go         # Embedding logic
│   ├── search.go        # Cosine search logic
│   ├── types.go
│   └── utils.go
├── ask.ts               # Takes top chunks, prompts LLM
├── summarizer.ts        # Prompt building logic
├── data/
│   ├── chunked.jsonl
│   └── .index.json

```

## 🧠 What I Learned

### Go Struct Tags

Tag format : `FieldName FieldType json:"jsonKeyName"`

- maps a Go field to a JSON key
- ensures compatibility even if names dont match exactly

Example: `Name string json:"user_name"` // maps JSON "user_name" to Go field Name`

### Vector Embeddings

This project deepened my understanding of how vector embeddings power semantic search and LLM-based tools:

- **Embeddings represent meaning** — I learned that OpenAI's `text-embedding-ada-002` model converts text/code into high-dimensional vectors that capture semantic relationships, not just keywords.
- **Chunks of code become coordinates** — Each code snippet is embedded as a 1536-dimensional vector, placing it in a conceptual space where similar functionality or intent lands nearby.
  > You can think of an embedding as the GPS coordinates of a thought in a high-dimensional space.
- **Semantic search enables smart retrieval** — Instead of relying on exact string matches, embeddings let us retrieve code that's relevant to a question based on meaning.
- **This mirrors real-world tools** — This is the same foundation used by tools like Sourcegraph Cody, GitHub Copilot, and ChatGPT when answering questions about large codebases.

Through building this pipeline in Go, I not only integrated an OpenAI API into a real-world tool but also came away with a stronger grasp of how LLMs and embeddings work together in production systems.
