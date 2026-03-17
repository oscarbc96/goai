import { readFileSync } from "fs";
import { join } from "path";

const fixturesDir = join(import.meta.dir, "..", "fixtures");
const streamData = readFileSync(join(fixturesDir, "stream_100x500.jsonl"));
const singleData = readFileSync(join(fixturesDir, "generate_single.json"));

// Mock server serves Chat Completions API format.
// Both Go and TS benchmarks use this format for fair comparison
// (same SSE fixtures, same parse workload).
export function startMockServer(): { url: string; stop: () => void } {
  const server = Bun.serve({
    port: 0,
    fetch(req) {
      const url = new URL(req.url);

      if (url.pathname === "/v1/chat/completions") {
        return req.json().then((body: any) => {
          if (body.stream) {
            return new Response(streamData, {
              headers: {
                "Content-Type": "text/event-stream",
                "Cache-Control": "no-cache",
                Connection: "keep-alive",
              },
            });
          }
          return new Response(singleData, {
            headers: { "Content-Type": "application/json" },
          });
        });
      }
      return new Response("not found", { status: 404 });
    },
  });

  return {
    url: `http://localhost:${server.port}`,
    stop: () => server.stop(),
  };
}
