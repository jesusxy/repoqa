import { ChatMessage, ScoredChunk } from "./types";

export function buildPrompt(
  query: string,
  chunks: ScoredChunk[]
): ChatMessage[] {
  const systemMessage: ChatMessage = {
    role: "system",
    content:
      "You are a senior engineer answering questions about a codebase. Use only the provided code snippets to answer. If code reveals config, structure, or usage patterns, summarize those clearly. Do not guess or include general explanations unless they are directly inferred from the code.",
  };

  let prompt = `Question: ${query}\n\nRelevant Code:\n\n`;

  for (const chunk of chunks) {
    // append the file + code sections to the prompt string.
    prompt += `\n---\nFile: ${chunk.file}\n${chunk.code}\n`;
  }

  const userMessage: ChatMessage = {
    role: "user",
    content: prompt,
  };

  return [systemMessage, userMessage];
}
