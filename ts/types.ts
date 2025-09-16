export type ScoredChunk = {
  id: string;
  file: string;
  code: string;
  score: Float32Array;
};

export type ChatMessage = {
  role: "system" | "user";
  content: string;
};

export type Chunk = Omit<ScoredChunk, "score"> & {
  endLine?: number;
  language?: string;
  startLine?: number;
  sourceType?: "ast" | "line"; // how it was chunked
  symbol?: string;
  type?: string;
};
