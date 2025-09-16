#!/usr/bin/env ts-node
import * as fs from "fs";
import * as path from "path";
import { Chunk } from "./types";
import Parser from "tree-sitter";
import JavaScript from "tree-sitter-javascript";
import TypeScript from "tree-sitter-typescript";
import Go from "tree-sitter-go";

const targetPath = process.argv[2];

const validFiles: Array<string> = ["go", "ts", "js", "py"];
if (!targetPath) {
  console.error(
    "❌ You must provide path to index.\nUsage: ts-node indexer.ts <path_to_index>"
  );
}

(async () => {
  try {
    console.log(`📇 Indexing: ${targetPath}`);
    const dataDir = path.join(__dirname, "../data");
    fs.mkdirSync(dataDir, { recursive: true });
    const stream = fs.createWriteStream(path.join(dataDir, "chunked.jsonl"));

    for await (const p of walk(targetPath)) {
      if (shouldSkip(p)) {
        console.log(`⏭️ Skipping: ${p}`);
        continue;
      }

      // only check for files we care about .go, .ts, .js, .py
      const ext = path.extname(p).slice(1);
      if (validFiles.includes(ext)) {
        try {
          // read file contents
          const fContents = fs.readFileSync(p, "utf-8");
          const chunked = chunkFile(fContents, p);

          for (const chunk of chunked) {
            const length = chunk.code.trim().length;
            console.log(
              `[indexing] ${chunk.file} - chunk ${chunk.id} - ${length} chars`
            );

            if (length > 0) {
              stream.write(JSON.stringify(chunk) + "\n");
            } else {
              console.log(`⚠️ Skipped empty chunk from ${chunk.file}`);
            }
          }
        } catch (e) {
          console.error(`❌ Error processing ${p}:`, e);
        }
      }
    }

    stream.close();
  } catch (err) {
    console.error(`❌ Error: `, err);
    process.exit(1);
  }
})();

async function* walk(dir: string): AsyncGenerator<string> {
  for await (const d of await fs.promises.opendir(dir)) {
    const entry = path.join(dir, d.name);
    if (d.isDirectory()) {
      if (shouldSkipDir(entry)) continue;
      yield* walk(entry);
    } else if (d.isFile()) {
      yield entry;
    }
  }
}

function shouldSkip(filePath: string): boolean {
  const filename = path.basename(filePath).toLowerCase();

  return (
    filePath.includes("__tests__") ||
    filePath.includes(".test.") ||
    filePath.includes(".spec.") ||
    filePath.includes("/.git/") ||
    filePath.includes("/node_modules/") ||
    filePath.includes("/dist/") ||
    filePath.includes("/coverage/") ||
    filePath.endsWith(".md") ||
    filePath.endsWith(".json") ||
    filePath.endsWith(".lock") ||
    filePath.endsWith(".snap") ||
    filename.endsWith(".d.ts") ||
    filename.endsWith(".config.js")
  );
}

function shouldSkipDir(dirPath: string): boolean {
  const ignoredDirs = [
    ".git",
    "node_modules",
    "dist",
    "coverage",
    "test",
    "__tests__",
  ];
  return ignoredDirs.some((name) => dirPath.endsWith(name));
}

///////////////// --------------------- PARSER ------------------------ ////////////////////

const tsParser = new Parser();
tsParser.setLanguage(TypeScript.typescript);

const jsParser = new Parser();
jsParser.setLanguage(JavaScript.language);

const goParser = new Parser();
goParser.setLanguage(Go.language);

const parserRegistry = {
  go: {
    parser: goParser,
    nodes: ["function_declaration", "method_declaration"],
  },
  js: {
    parser: jsParser,
    nodes: [
      "function_declaration",
      "function_expression",
      "arrow_function",
      "method_definition",
      "method_declaration",
      "class_declaration",
    ],
  },
  ts: {
    parser: tsParser,
    nodes: ["function_declaration", "method_definition", "class_declaration"],
  },
};

function extractChunksFromAST(
  language: "go" | "ts" | "js",
  code: string,
  filepath: string
) {
  const { parser, nodes } = parserRegistry[language];
  console.log(
    "[debug] parsing file with lang:",
    language,
    "chars:",
    code.length
  );
  const cleanedCode = sanitizeCodeForParser(code, language);
  const tree = parser.parse(cleanedCode);
  const root = tree.rootNode;

  const functions = root.descendantsOfType(nodes);

  if (!functions || functions.length === 0) {
    console.log(`⚠️ No chunkable nodes found in ${filepath}`);
    return [];
  }

  return functions.map((node, i) => ({
    id: `chunk_${i}`,
    code: code.slice(node.startIndex, node.endIndex),
    endLine: node.endPosition.row + 1,
    file: filepath,
    startLine: node.startPosition.row + 1,
    symbol: extractSymbolName(node) || `anonymous_${i}`,
    type: node.type,
  }));
}

function detectLanguage(filePath: string): "ts" | "go" | "js" | null {
  if (filePath.endsWith(".ts") || filePath.endsWith(".tsx")) return "ts";
  if (filePath.endsWith(".js")) return "js";
  if (filePath.endsWith(".go")) return "go";
  return null;
}

function chunkFile(contents: string, filePath: string): Array<Chunk> {
  const lang = detectLanguage(filePath);
  if (lang && parserRegistry[lang]) {
    try {
      return extractChunksFromAST(lang, contents, filePath);
    } catch (err) {
      console.error(`❌ AST chunking failed for ${filePath}:`, err);
      return []; // fallback
    }
  }

  // const LINES_PER_CHUNK = 80;
  // let result: Array<Chunk> = [];

  // const lines = contents.split("\n");

  // for (let i = 0; i < lines.length; i += LINES_PER_CHUNK) {
  //   const chunkedLines = lines.slice(i, i + LINES_PER_CHUNK).join("\n");

  //   result.push({
  //     id: `chunk_${Math.floor(i / LINES_PER_CHUNK) + 1}`,
  //     file: filePath,
  //     code: chunkedLines,
  //   });
  // }

  // return result;
}

function extractSymbolName(node: Parser.SyntaxNode): string | undefined {
  switch (node.type) {
    case "function_declaration":
    case "method_definition":
    case "class_declaration":
    case "method_declaration":
    case "function_definition": {
      const identifier = node.childForFieldName("name");
      return identifier?.text;
    }
    default:
      return undefined;
  }
}

function sanitizeCodeForParser(code: string, lang: "js" | "ts" | "go"): string {
  let cleaned = code;

  if (lang === "js" || lang === "ts") {
    cleaned = cleaned
      .replace(/^#!.*/, "") // remove shebang
      .replace(/[\uD800-\uDFFF]/g, ""); // remove problematic unicode
  }

  // Normalize newlines for all languages (safe)
  cleaned = cleaned.replace(/\r\n/g, "\n").replace(/\r/g, "\n");

  return cleaned;
}
