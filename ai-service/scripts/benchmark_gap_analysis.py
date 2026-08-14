"""Checklist 13.1: benchmark gap-analysis speed against the PRD's stated
target -- run against the REAL local Ollama, not a mock (see
tests/test_gap_analysis_llm.py for the mocked correctness tests; this
measures wall-clock latency instead).

No gap-analysis-specific target exists anywhere in the PRD docs
(checked: graphify-out/converted/*.md, not assumed) -- the closest
stated proxy is edugraph-architecture.docx's SLO table: "AI job
completion (study plan): 95% of jobs complete in < 30 seconds."
Gap analysis and study plan generation are architecturally identical
(both single LLM calls dispatched via generate_with_fallback, off a
Redis-queued job, see app/utils/llm_provider.py's docstring), so this
uses that same 30s/P95 bar as the working target, explicitly flagged as
a proxy rather than a number the PRD actually states for this specific
feature.

Usage (from ai-service/, with the venv active and Ollama running with
OLLAMA_MODEL pulled -- see .env.example):
    python scripts/benchmark_gap_analysis.py [--runs N]
"""

from __future__ import annotations

import argparse
import asyncio
import statistics
import time

from app.services.gap_analysis.llm import synthesize_insights

TARGET_P95_SECONDS = 30.0  # see module docstring on why this is a proxy, not a stated gap-analysis target

# Representative of a real attempt: MAX_GAPS_TO_EXPLAIN (gap_analysis/
# llm.py) caps at 10, and this uses that ceiling deliberately -- the
# worst case (a genuinely bad exam) is the case that matters for a
# latency benchmark, not a light 1-2-gap attempt that isn't
# representative of what makes this pipeline slow.
def _sample_gap_contexts(n: int = 10) -> list[dict]:
    contexts = []
    for i in range(1, n + 1):
        contexts.append(
            {
                "index": i,
                "question_text": f"Sample question {i} testing a moderately complex Biology concept, "
                "long enough to be representative of a real exam question's token count.",
                "symptom_topic": f"Topic {i}",
                "root_cause_topic": f"Prerequisite Topic {i}" if i % 2 == 0 else None,
                "root_cause_grade": 8 if i % 2 == 0 else None,
                "severity": 0.3 + (i % 5) * 0.1,
            }
        )
    return contexts


async def _run_once() -> float:
    start = time.perf_counter()
    explanations, summary, model = await synthesize_insights(
        exam_title="Grade 9 Biology Unit Test",
        exam_scope="unit_test",
        subject_code="BIO",
        grade_level=9,
        percentage=42.0,
        gap_contexts=_sample_gap_contexts(),
    )
    elapsed = time.perf_counter() - start
    status = "ok" if explanations or summary else "FAILED (no provider produced output)"
    print(f"  run took {elapsed:6.2f}s -- model={model} status={status}")
    return elapsed


async def main(runs: int) -> None:
    print(f"Benchmarking synthesize_insights against the real configured LLM provider ({runs} runs, 10 gaps each)...")
    print("This calls a real LLM -- see app/core/config.py's LLM_PROVIDER/OLLAMA_HOST/OLLAMA_MODEL for what's targeted.\n")

    timings = [await _run_once() for _ in range(runs)]

    timings.sort()
    p50 = statistics.median(timings)
    p95_index = min(len(timings) - 1, int(len(timings) * 0.95))
    p95 = timings[p95_index]

    print(f"\nP50: {p50:.2f}s   P95: {p95:.2f}s   min: {timings[0]:.2f}s   max: {timings[-1]:.2f}s")
    print(f"Proxy target (study plan's stated SLO, see module docstring): P95 < {TARGET_P95_SECONDS:.0f}s")
    if p95 < TARGET_P95_SECONDS:
        print(f"PASS -- P95 ({p95:.2f}s) is within the proxy target.")
    else:
        print(f"MISS -- P95 ({p95:.2f}s) exceeds the proxy target by {p95 - TARGET_P95_SECONDS:.2f}s.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--runs", type=int, default=5, help="number of benchmark iterations (default: 5)")
    args = parser.parse_args()
    asyncio.run(main(args.runs))
