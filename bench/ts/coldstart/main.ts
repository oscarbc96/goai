// Cold start: measures import + first call latency.
// Each invocation is a separate process for accurate measurement.
import { generateText } from "ai";
import { createOpenAI } from "@ai-sdk/openai";

const start = Bun.nanoseconds();

// Inline mock server returning Chat Completions format.
const server = Bun.serve({
  port: 0,
  fetch() {
    return new Response(
      JSON.stringify({
        id: "chatcmpl-bench",
        object: "chat.completion",
        created: 1700000000,
        model: "gpt-4o",
        choices: [
          {
            index: 0,
            message: { role: "assistant", content: "ok" },
            finish_reason: "stop",
          },
        ],
        usage: {
          prompt_tokens: 1,
          completion_tokens: 1,
          total_tokens: 2,
        },
      }),
      { headers: { "Content-Type": "application/json" } }
    );
  },
});

const provider = createOpenAI({
  baseURL: `http://localhost:${server.port}/v1`,
  apiKey: "bench",
});
// Use .chat() for Chat Completions API.
await generateText({ model: provider.chat("gpt-4o"), prompt: "hi" });

const elapsed = Bun.nanoseconds() - start;
console.log(JSON.stringify({ benchmark: "cold_start", ns: elapsed }));

server.stop();
