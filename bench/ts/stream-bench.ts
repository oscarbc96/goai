import { Bench } from "tinybench";
import { streamText, generateText } from "ai";
import { createOpenAI } from "@ai-sdk/openai";
import { startMockServer } from "./mock-server";

const { url, stop } = startMockServer();
// Use .chat() to force Chat Completions API (not Responses API).
// Both Go and TS benchmarks use Chat Completions for fair comparison.
const provider = createOpenAI({ baseURL: url + "/v1", apiKey: "bench" });
const model = provider.chat("gpt-4o");

// --- Streaming throughput ---
const throughputBench = new Bench({ iterations: 200, warmupIterations: 10 });
throughputBench.add("streaming_throughput", async () => {
  const result = streamText({ model, prompt: "bench" });
  for await (const _ of result.textStream) {
    // consume all chunks
  }
});

await throughputBench.run();
for (const task of throughputBench.tasks) {
  const r = task.result!;
  console.log(
    JSON.stringify({
      benchmark: "streaming_throughput",
      ns_per_op: Math.round(r.mean * 1e6), // ms → ns
      ops_per_sec: Math.round(1000 / r.mean),
      allocs_per_op: 0,
      bytes_per_op: 0,
    })
  );
}

// --- Time to first chunk ---
// Warmup: 10 iterations to stabilize JIT/caches (matching throughput bench).
for (let i = 0; i < 10; i++) {
  const result = streamText({ model, prompt: "bench" });
  for await (const _ of result.textStream) {
    break;
  }
  for await (const _ of result.textStream) {
  }
}

const ttfcRuns = 200;
const ttfcDurations: number[] = [];
for (let i = 0; i < ttfcRuns; i++) {
  const start = Bun.nanoseconds();
  const result = streamText({ model, prompt: "bench" });
  for await (const _ of result.textStream) {
    ttfcDurations.push(Bun.nanoseconds() - start);
    break;
  }
  // Drain remaining outside measurement.
  for await (const _ of result.textStream) {
  }
}
ttfcDurations.sort((a, b) => a - b);
console.log(
  JSON.stringify({
    benchmark: "time_to_first_chunk",
    median_ns: Math.round(ttfcDurations[Math.floor(ttfcDurations.length / 2)]),
    p99_ns: Math.round(
      ttfcDurations[Math.floor(ttfcDurations.length * 0.99)]
    ),
  })
);

// --- GenerateText (non-streaming) ---
const genBench = new Bench({ iterations: 200, warmupIterations: 10 });
genBench.add("generate_text", async () => {
  await generateText({ model, prompt: "bench" });
});

await genBench.run();
for (const task of genBench.tasks) {
  const r = task.result!;
  console.log(
    JSON.stringify({
      benchmark: "generate_text",
      ns_per_op: Math.round(r.mean * 1e6),
      ops_per_sec: Math.round(1000 / r.mean),
    })
  );
}

stop();
