"""
Real prediction-quality metrics (EG-GCKT Milestone 10, spec section 17):
AUC, log loss, Brier score, expected calibration error. No sklearn
dependency (none exists in this codebase's requirements.txt) -- each is a
small, standard, independently-verifiable implementation.
"""

from __future__ import annotations

import math

_EPS = 1e-9


def _clip(p: float) -> float:
    return min(1.0 - _EPS, max(_EPS, p))


def auc(y_true: list[float], y_prob: list[float]) -> float | None:
    """ROC AUC via the Mann-Whitney U statistic: the probability a
    randomly-chosen positive is ranked above a randomly-chosen negative.
    None when the labels are all one class (AUC is undefined without both
    classes present -- not a bug to paper over with a fake 0.5)."""
    pairs = sorted(zip(y_prob, y_true), key=lambda p: p[0])
    n = len(pairs)
    ranks = [0.0] * n
    i = 0
    while i < n:
        j = i
        while j < n and pairs[j][0] == pairs[i][0]:
            j += 1
        avg_rank = (i + 1 + j) / 2.0  # 1-indexed average rank for ties
        for k in range(i, j):
            ranks[k] = avg_rank
        i = j

    n_pos = sum(1 for _, y in pairs if y > 0.5)
    n_neg = n - n_pos
    if n_pos == 0 or n_neg == 0:
        return None

    rank_sum_pos = sum(r for r, (_, y) in zip(ranks, pairs) if y > 0.5)
    u_statistic = rank_sum_pos - n_pos * (n_pos + 1) / 2.0
    return u_statistic / (n_pos * n_neg)


def log_loss(y_true: list[float], y_prob: list[float]) -> float:
    """Standard binary cross-entropy, mean over instances."""
    n = len(y_true)
    total = 0.0
    for y, p in zip(y_true, y_prob):
        p = _clip(p)
        total += -(y * math.log(p) + (1 - y) * math.log(1 - p))
    return total / n if n else float("nan")


def brier_score(y_true: list[float], y_prob: list[float]) -> float:
    """Mean squared error between predicted probability and the binary
    outcome -- a proper scoring rule, unlike raw accuracy."""
    n = len(y_true)
    return sum((p - y) ** 2 for y, p in zip(y_true, y_prob)) / n if n else float("nan")


def expected_calibration_error(y_true: list[float], y_prob: list[float], n_bins: int = 10) -> float:
    """Standard ECE: bin predictions by predicted probability, compare
    each bin's average predicted probability against its actual observed
    positive rate, weight by bin size."""
    n = len(y_true)
    if n == 0:
        return float("nan")

    bins: list[list[tuple[float, float]]] = [[] for _ in range(n_bins)]
    for y, p in zip(y_true, y_prob):
        idx = min(n_bins - 1, int(p * n_bins))
        bins[idx].append((y, p))

    ece = 0.0
    for bucket in bins:
        if not bucket:
            continue
        avg_true = sum(y for y, _ in bucket) / len(bucket)
        avg_pred = sum(p for _, p in bucket) / len(bucket)
        ece += (len(bucket) / n) * abs(avg_true - avg_pred)
    return ece
