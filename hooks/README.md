# Vigil Hooks - Development Guide

Hooks are scripts that run after every successful capture. Vigil passes the captured image path as the first argument and stores whatever the script prints to stdout in the database. This is how analysis metrics, alerts, and integrations work.

## The Contract

Your hook receives **one argument**: the absolute path to the captured JPEG.

```
hooks/my-hook.py /data/captures/2026-02-17/14-30-00.jpg
```

| Behavior | What Vigil Does |
|----------|----------------|
| **stdout** | Stored in the database as `hook_output`. Shown on the dashboard. Return JSON for structured data. |
| **stderr** | Appended to stdout with `--- stderr ---` separator. Use for debug logging. |
| **Exit 0** | Marked as `hook_status: "success"` |
| **Exit non-zero** | Marked as `hook_status: "error"` |
| **No hook configured** | `hook_status: "skipped"` - no script runs |
| **Timeout (60s default)** | Process killed, marked as `error` |

## Language Support

Vigil auto-detects the interpreter by file extension:

| Extension | Invoked as |
|-----------|-----------|
| `.py` | `python <script> <image>` (prefers `.venv` if present) |
| `.rb` | `ruby <script> <image>` |
| `.js` | `node <script> <image>` |
| anything else | `<script> <image>` (must be executable) |

## Local Development

### Python (Recommended)

```bash
# From the project root
python -m venv .venv

# Windows
.\.venv\Scripts\activate

# Linux/macOS
source .venv/bin/activate

# Install hook dependencies
pip install -r hooks/requirements.txt

# Test your hook directly
python hooks/analyze.py data/captures/2026-02-17/14-30-00.jpg
```

> **Tip:** Vigil automatically finds `.venv/Scripts/python.exe` (Windows) or `.venv/bin/python3` (Linux). You don't need to configure anything - just create the venv in the project root.

### Testing Independently

Hooks are plain scripts. You can test them without running Vigil:

```bash
# Should print JSON to stdout and exit 0
python hooks/analyze.py path/to/any/image.jpg
echo $?   # Should be 0
```

If your hook has errors, check stderr:
```bash
python hooks/analyze.py path/to/image.jpg 2>hook_errors.log
cat hook_errors.log
```

### Docker Development

The Docker image bakes in `hooks/` at build time. To iterate without rebuilding:

```yaml
# docker-compose.yml
volumes:
  - ./hooks:/home/vigil/hooks
```

Edit locally, trigger a capture from the web UI, check logs with `docker compose logs -f`.

## Writing a New Hook

### Minimal Example

```python
#!/usr/bin/env python3
"""My custom Vigil hook."""
import json
import sys

image_path = sys.argv[1]

# Do something with the image...
result = {"status": "ok", "path": image_path}

# Print JSON to stdout - this gets stored in the DB and shown on the dashboard
print(json.dumps(result))
```

### Best Practices

1. **Always output valid JSON.** The dashboard parses `hook_output` as JSON to show metrics. If your output isn't JSON, it shows as raw text.

2. **Handle missing/corrupt images gracefully.** The image might be empty or unreadable. Return an error JSON rather than crashing:
   ```python
   if not os.path.exists(sys.argv[1]):
       print(json.dumps({"error": "file not found"}))
       sys.exit(1)
   ```

3. **Keep it fast.** The capture pipeline waits for your hook (up to 60s timeout). Heavy processing blocks the next scheduled capture. If you need to do slow work, have your hook enqueue it and return immediately.

4. **Use stderr for debug logging.** Stdout is the "return value" - don't mix log messages into it. Use `print("...", file=sys.stderr)` for diagnostics.

5. **Pin your dependencies.** Add them to `hooks/requirements.txt`. The Docker image installs these at build time via `pip install -r hooks/requirements.txt`.

6. **Don't modify the image.** The hook receives the capture path but should treat it as read-only. If you need to create derived files (thumbnails, annotated copies), write them elsewhere.

7. **Don't assume the working directory.** Always use the absolute path passed as the argument. Don't assume `os.getcwd()` is the project root.

### Gotchas Learned the Hard Way

- **opencv-python vs opencv-python-headless:** In Docker or on headless servers (like a Pi), use `opencv-python-headless`. The full `opencv-python` package pulls in Qt/GTK dependencies that don't exist in the container.

- **numpy version conflicts:** Pin minimum versions in `requirements.txt` but don't over-constrain. Let pip resolve compatible versions.

- **File comparison across days:** `analyze.py` compares against the "previous capture in the same directory." Since directories are date-based (`captures/2026-02-17/`), the first capture of a new day has nothing to compare against - handle this edge case.

- **Pi 3B+ performance:** OpenCV operations on a Pi 3B+ are slow. Resize images before heavy processing (the included `analyze.py` uses `COMPARE_MAX_DIM = 640` for this reason).

## Included Hooks

### `analyze.py`

Image quality analysis and motion detection. Outputs:

```json
{
  "brightness": 142.3,
  "sharpness": 1250.7,
  "resolution": "1280x960",
  "file_size_kb": 138.3,
  "change_pct": 12.4,
  "motion_detected": true,
  "timestamp": "2026-02-17T17:30:00+00:00"
}
```

**Dependencies:** `opencv-python-headless`, `Pillow`, `numpy` (see `requirements.txt`)

**Enable it:** Set `hook_path` to `hooks/analyze.py` on the Config page in the web UI.
