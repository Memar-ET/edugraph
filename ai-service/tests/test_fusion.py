"""Tests for app/services/knowledge_tracing/fusion.py's pure weighting/
status/trend helpers -- checklist requirement "unit-test fusion weighting
and provenance retention."
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

from app.services.knowledge_tracing.fusion import (
    RECENCY_HALF_LIFE_DAYS,
    _mastery_status,
    _recency_weight,
    _source_weight,
    _trend,
)


def test_recency_weight_is_one_for_brand_new_evidence() -> None:
    assert abs(_recency_weight(datetime.now(timezone.utc)) - 1.0) < 1e-6


def test_recency_weight_halves_at_the_half_life() -> None:
    old = datetime.now(timezone.utc) - timedelta(days=RECENCY_HALF_LIFE_DAYS)
    assert abs(_recency_weight(old) - 0.5) < 1e-2


def test_recency_weight_decreases_monotonically_with_age() -> None:
    now = datetime.now(timezone.utc)
    ages = [0, 5, 30, 90, 365]
    weights = [_recency_weight(now - timedelta(days=a)) for a in ages]
    assert weights == sorted(weights, reverse=True)


def test_source_weight_scales_with_sample_size_below_minimum() -> None:
    now = datetime.now(timezone.utc)
    low_sample = _source_weight("bkt", reliability=0.8, sample_size=1, created_at=now)
    high_sample = _source_weight("bkt", reliability=0.8, sample_size=10, created_at=now)
    assert low_sample < high_sample


def test_source_weight_falls_back_to_default_reliability_table_when_row_reliability_missing() -> None:
    now = datetime.now(timezone.utc)
    weight = _source_weight("graph_reasoning", reliability=None, sample_size=5, created_at=now)
    assert weight > 0  # uses DEFAULT_SOURCE_RELIABILITY["graph_reasoning"], not zero


def test_source_weight_uses_conservative_fallback_for_unrecognized_provenance() -> None:
    now = datetime.now(timezone.utc)
    known = _source_weight("bkt", reliability=None, sample_size=5, created_at=now)
    unknown = _source_weight("some_future_engine", reliability=None, sample_size=5, created_at=now)
    assert unknown < known  # unrecognized source gets the conservative FALLBACK_RELIABILITY


def test_mastery_status_thresholds() -> None:
    assert _mastery_status(None) == "unknown"
    assert _mastery_status(0.1) == "emerging"
    assert _mastery_status(0.5) == "proficient"
    assert _mastery_status(0.9) == "mastered"


def test_trend_classification() -> None:
    assert _trend(0.5, None) is None  # no prior estimate -- nothing to compare
    assert _trend(0.6, 0.5) == "improving"
    assert _trend(0.4, 0.5) == "declining"
    assert _trend(0.501, 0.5) == "stable"  # within TREND_EPSILON
