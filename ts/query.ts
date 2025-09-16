import { ScoredChunk } from "./types";
import { exec } from "child_process";
import { promisify } from "util";

const execAsync = promisify(exec);

export async function getTopKChunks(
  query: string,
  topK: number
): Promise<ScoredChunk[]> {
  try {
    const { stdout } = await execAsync(
      `./repoqa query --json --top ${topK} ${query}`
    );

    const chunks: ScoredChunk[] = JSON.parse(stdout);

    if (!Array.isArray(chunks)) {
      throw new Error("❌ Invalid output: expected an array of scored chunks");
    }

    return chunks;
  } catch (e) {
    throw new Error(`Failed to get relevant chunks: ${(e as Error).message}`);
  }
}
