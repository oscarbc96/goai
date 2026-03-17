#!/usr/bin/env bash
# run_bench.sh - run Go benchmarks and produce JSONL output.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/../results"
mkdir -p "$RESULTS_DIR"
OUTPUT="$RESULTS_DIR/go.jsonl"

cd "$SCRIPT_DIR"

echo "Running Go benchmarks..."

# Run GoAI-related benchmarks only.
GOWORK=off go test -bench='Benchmark(Streaming|TimeToFirst|Concurrent|Memory|Overhead|Schema)' \
    -benchmem -benchtime=500x -count=3 -run='^$' . 2>&1 | tee /tmp/goai_bench_raw.txt

echo "Parsing results..."

# Parse go test -bench output to JSONL using python (more robust than bash).
python3 << PYEOF
import re, json, sys
from collections import defaultdict

NAME_MAP = {
    "BenchmarkStreamingThroughput": "streaming_throughput",
    "BenchmarkTimeToFirstChunk": "time_to_first_chunk",
    "BenchmarkConcurrentStreams": "concurrent_streams",
    "BenchmarkMemory1Stream": "memory_1_stream",
    "BenchmarkMemory10Streams": "memory_10_streams",
    "BenchmarkMemory50Streams": "memory_50_streams",
    "BenchmarkMemory100Streams": "memory_100_streams",
    "BenchmarkOverheadRawHTTP": "raw_http",
    "BenchmarkOverheadGoAI": "goai_sdk",
    "BenchmarkOverheadGenerateText": "generate_text",
    "BenchmarkSchemaSimple": "schema_simple",
    "BenchmarkSchemaComplex": "schema_complex",
}

by_name = defaultdict(list)

with open("/tmp/goai_bench_raw.txt") as f:
    for line in f:
        if not line.startswith("Benchmark"):
            continue
        # Extract benchmark name (strip -N suffix)
        name = re.match(r"(\w+)-\d+", line)
        if not name:
            continue
        name = name.group(1)
        if name not in NAME_MAP:
            continue

        bench = NAME_MAP[name]
        entry = {"benchmark": bench}

        # Extract ns/op
        m = re.search(r"([\d.]+)\s+ns/op", line)
        if m:
            entry["ns_per_op"] = int(round(float(m.group(1))))
            entry["ops_per_sec"] = int(1e9 / entry["ns_per_op"]) if entry["ns_per_op"] > 0 else 0

        # Extract B/op
        m = re.search(r"(\d+)\s+B/op", line)
        if m:
            entry["bytes_per_op"] = int(m.group(1))

        # Extract allocs/op
        m = re.search(r"(\d+)\s+allocs/op", line)
        if m:
            entry["allocs_per_op"] = int(m.group(1))

        # Extract heap-bytes/op (custom metric for memory benchmarks)
        m = re.search(r"([\d.]+)\s+heap-bytes/op", line)
        if m:
            entry["heap_bytes"] = int(round(float(m.group(1))))

        by_name[bench].append(entry)

# Take median of 3 runs.
results = []
for name, entries in by_name.items():
    if len(entries) == 1:
        results.append(entries[0])
    elif "ns_per_op" in entries[0]:
        entries.sort(key=lambda x: x["ns_per_op"])
        results.append(entries[len(entries) // 2])
    else:
        results.append(entries[-1])

outpath = "$OUTPUT"
with open(outpath, "w") as f:
    for r in results:
        f.write(json.dumps(r) + "\n")

print(f"Parsed {len(results)} benchmarks to {outpath}")
PYEOF
python3 -c "
import sys, json
with open(sys.argv[1]) as f:
    lines = f.readlines()
# Just a validation pass
for line in lines:
    json.loads(line)
print(f'Validated {len(lines)} results')
" "$OUTPUT" || true

# Run cold start benchmark.
echo "Running cold start benchmark..."
cd "$SCRIPT_DIR/coldstart"
GOWORK=off go build -o coldstart .
COLD_RESULTS=()
for i in $(seq 20); do
    ns=$(./coldstart | jq -r '.ns')
    COLD_RESULTS+=("$ns")
done
rm -f coldstart

# Sort and compute median/p99.
SORTED=($(printf '%s\n' "${COLD_RESULTS[@]}" | sort -n))
COUNT=${#SORTED[@]}
MEDIAN=${SORTED[$((COUNT / 2))]}
P99_IDX=$(echo "scale=0; $COUNT * 99 / 100" | bc)
P99=${SORTED[$P99_IDX]}

echo "{\"benchmark\":\"cold_start\",\"median_ns\":$MEDIAN,\"p99_ns\":$P99}" >> "$OUTPUT"

echo ""
echo "Go results written to $OUTPUT"
cat "$OUTPUT"
