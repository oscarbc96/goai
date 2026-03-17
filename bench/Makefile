.PHONY: bench-go bench-ts bench-all bench-3x report fixtures clean

# Generate fixture files.
fixtures:
	cd fixtures && go run gen.go

# Run Go benchmarks → results/go.jsonl
bench-go:
	cd go && bash run_bench.sh

# Run TS benchmarks → results/ts.jsonl
bench-ts:
	cd ts && bun run bench:stream > ../results/ts-stream.jsonl
	cd ts && bun run bench:memory >> ../results/ts-stream.jsonl
	cd ts && bun run bench:schema >> ../results/ts-stream.jsonl
	cd ts && bun run bench:cold >> ../results/ts-stream.jsonl
	mv results/ts-stream.jsonl results/ts.jsonl
	@echo ""
	@echo "TS results written to results/ts.jsonl"
	@cat results/ts.jsonl

# Run both and generate report.
bench-all: bench-go bench-ts report

# Generate comparison report from results.
report:
	bash collect.sh

# Run 3 independent runs and generate averaged RESULTS.md.
bench-3x:
	bash collect.sh --3x

# Regenerate RESULTS.md from existing run files (no re-bench).
bench-3x-report:
	bash collect.sh --3x-report

clean:
	rm -rf results/*.jsonl results/REPORT.md
	rm -f go/coldstart/coldstart
