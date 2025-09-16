#!/usr/bin/env ts-node
import { OpenAI } from "openai";
import "dotenv/config";
import { getTopKChunks } from "./query";
import { buildPrompt } from "./summarizer";
import { ScoredChunk } from "./types";

const query = process.argv.slice(2).join("");

if (!query) {
  console.error(
    '❌ You must provide a query.\nUsage: ts-node ask.ts "What does the parser do?"'
  );
  process.exit(1);
}

(async () => {
  try {
    console.log(`🧠 Query: "${query}"`);
    const topK = 3;

    const chunks: ScoredChunk[] = await getTopKChunks(query, topK);
    console.log(`📎 Found ${chunks.length} relevant chunks`);

    const prompt = buildPrompt(query, chunks);

    const openAi = new OpenAI({ apiKey: process.env.OPENAI_API_KEY });

    const completion = await openAi.chat.completions.create({
      model: "gpt-4",
      messages: prompt,
    });

    console.log("\n🧠 Answer:\n");
    console.log(completion.choices[0].message.content);
  } catch (err) {
    console.error(`❌ Error: ${err.message}`);
    process.exit(1);
  }
})();
