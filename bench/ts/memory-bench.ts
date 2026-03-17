import { streamText } from "ai";
import { createOpenAI } from "@ai-sdk/openai";
import { startMockServer } from "./mock-server";

const { url, stop } = startMockServer();
// Use .chat() for Chat Completions API (matching Go benchmark).
const provider = createOpenAI({ baseURL: url + "/v1", apiKey: "bench" });
const model = provider.chat("gpt-4o");

async function measureMemory(streamCount: number): Promise<number> {
  // Force GC before measurement.
  Bun.gc(true);
  const before = process.memoryUsage().heapUsed;

  // Create streams sequentially (matching Go benchmark).
  const streams: any[] = [];
  for (let i = 0; i < streamCount; i++) {
    const result = streamText({ model, prompt: "bench" });
    streams.push(result);
    // Read first chunk to establish stream.
    for await (const _ of result.textStream) {
      break;
    }
  }

  Bun.gc(true);
  const after = process.memoryUsage().heapUsed;

  // Drain all streams after measurement.
  for (const s of streams) {
    for await (const _ of s.textStream) {
    }
  }

  return Math.max(0, after - before);
}

for (const count of [1, 10, 50, 100]) {
  // Run 3 times and take median.
  const measurements: number[] = [];
  for (let i = 0; i < 3; i++) {
    measurements.push(await measureMemory(count));
  }
  measurements.sort((a, b) => a - b);
  const median = measurements[Math.floor(measurements.length / 2)];

  console.log(
    JSON.stringify({
      benchmark: `memory_${count}_stream${count > 1 ? "s" : ""}`,
      heap_bytes: median,
    })
  );
}

stop();
