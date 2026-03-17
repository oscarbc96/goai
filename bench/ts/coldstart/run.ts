// Runs coldstart/main.ts 20 times in separate processes and aggregates.
import { join } from "path";

const mainPath = join(import.meta.dir, "main.ts");
const results: number[] = [];

for (let i = 0; i < 20; i++) {
  const proc = Bun.spawn(["bun", "run", mainPath], {
    stdout: "pipe",
    stderr: "inherit",
  });
  const text = await new Response(proc.stdout).text();
  const data = JSON.parse(text.trim());
  results.push(data.ns);
}

results.sort((a, b) => a - b);
console.log(
  JSON.stringify({
    benchmark: "cold_start",
    median_ns: Math.round(results[Math.floor(results.length / 2)]),
    p99_ns: Math.round(results[Math.floor(results.length * 0.99)]),
  })
);
