#!/usr/bin/env python3
"""
Vigil Analysis Hook — Post-capture image analysis.

Usage:  python analyze.py <image_path>

Outputs a JSON object to stdout with image quality metrics and
change detection relative to the previous capture in the same
date directory.

Metrics:
  brightness     – Mean pixel luminance (0–255)
  sharpness      – Laplacian variance (higher = sharper)
  resolution     – "WxH"
  file_size_kb   – File size in KB
  change_pct     – % of pixels that changed vs previous capture (0–100)
  motion_detected– True if change_pct > threshold
  timestamp      – ISO 8601 analysis time
"""

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

import cv2
import numpy as np
from PIL import Image

# --- Configuration -----------------------------------------------------------

# Minimum % of pixels that must differ to count as motion.
MOTION_THRESHOLD = 5.0

# Per-pixel intensity difference threshold (0–255) to count a pixel as "changed".
PIXEL_DIFF_THRESHOLD = 30

# Maximum dimension to resize images to for comparison (speed optimization).
COMPARE_MAX_DIM = 640


# --- Analysis Functions ------------------------------------------------------

def calc_brightness(gray: np.ndarray) -> float:
    """Mean pixel brightness (0–255)."""
    return round(float(np.mean(gray)), 1)


def calc_sharpness(gray: np.ndarray) -> float:
    """Laplacian variance — higher means sharper."""
    lap = cv2.Laplacian(gray, cv2.CV_64F)
    return round(float(lap.var()), 1)


def calc_change(current_path: Path, gray: np.ndarray) -> tuple[float, bool]:
    """
    Compare current image to the previous capture in the same directory.
    Returns (change_pct, motion_detected).
    """
    parent = current_path.parent
    siblings = sorted(
        [f for f in parent.glob("*.jpg") if f != current_path],
        key=lambda f: f.name,
    )

    if not siblings:
        # No previous capture to compare against.
        return 0.0, False

    prev_path = siblings[-1]  # Most recent before current (sorted by name).

    prev_img = cv2.imread(str(prev_path), cv2.IMREAD_GRAYSCALE)
    if prev_img is None:
        return 0.0, False

    # Resize both to the same dimensions for comparison.
    h, w = gray.shape[:2]
    scale = min(1.0, COMPARE_MAX_DIM / max(h, w))
    if scale < 1.0:
        new_w, new_h = int(w * scale), int(h * scale)
        gray_small = cv2.resize(gray, (new_w, new_h))
        prev_small = cv2.resize(prev_img, (new_w, new_h))
    else:
        gray_small = gray
        prev_small = cv2.resize(prev_img, (w, h))

    diff = cv2.absdiff(gray_small, prev_small)
    changed_pixels = np.count_nonzero(diff > PIXEL_DIFF_THRESHOLD)
    total_pixels = diff.size
    change_pct = round((changed_pixels / total_pixels) * 100, 1)
    motion = bool(change_pct > MOTION_THRESHOLD)

    return change_pct, motion


# --- Main --------------------------------------------------------------------

def analyze(image_path: str) -> dict:
    """Run all analyses on the given image and return a metrics dict."""
    path = Path(image_path)

    if not path.exists():
        return {"error": f"File not found: {image_path}"}

    # Load image.
    img = cv2.imread(str(path))
    if img is None:
        return {"error": f"Could not decode image: {image_path}"}

    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    h, w = img.shape[:2]

    # File info.
    file_size_kb = round(path.stat().st_size / 1024, 1)

    # Compute metrics.
    brightness = calc_brightness(gray)
    sharpness = calc_sharpness(gray)
    change_pct, motion_detected = calc_change(path, gray)

    return {
        "brightness": brightness,
        "sharpness": sharpness,
        "resolution": f"{w}x{h}",
        "file_size_kb": file_size_kb,
        "change_pct": change_pct,
        "motion_detected": motion_detected,
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: analyze.py <image_path>"}))
        sys.exit(1)

    result = analyze(sys.argv[1])
    print(json.dumps(result))
