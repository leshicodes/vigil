# Vigil

A self-hosted time-lapse monitoring tool. Point it at a camera, set a schedule, and Vigil captures images on a cron, runs optional analysis hooks, and serves everything through a local dashboard.

Built with Go on the backend and React + TypeScript on the frontend. The whole thing compiles down to a single binary with the UI embedded, so deployment is just copying one file.

## Quick Start

### Docker (easiest)

```bash
docker compose up -d
```

That gives you a running instance at http://localhost:8080 with the mock camera. To use a real camera or enable auth, edit docker-compose.yml or pass environment variables:

```bash
docker run -d \
  -p 8080:8080 \
  -v ./data:/data \
  -e VIGIL_CAMERA=ffmpeg \
  -e VIGIL_API_KEY=your-secret-here \
  ghcr.io/leshicodes/vigil:latest
```

### From Source

You will need Go 1.22+, Node 20+, and optionally Python 3.10+ (for the analysis hook).

```bash
# Clone and build
git clone https://github.com/leshicodes/vigil.git
cd vigil

# Build the frontend
cd web && npm install && npx vite build && cd ..

# Build the Go binary
go build -o vigil .

# Run it
./vigil --data-dir=./data --camera=mock --static-dir=./web/dist
```

Open http://localhost:8080 and you should see the dashboard.

## Configuration

Vigil is configured through flags or environment variables. Flags take precedence.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--data-dir` | `VIGIL_DATA_DIR` | `./data` | Where captures and the database live |
| `--camera` | `VIGIL_CAMERA` | `mock` | Camera driver (see below) |
| `--port` | `VIGIL_PORT` | `8080` | HTTP server port |
| `--static-dir` | `VIGIL_STATIC_DIR` | `./web/dist` | Path to the built frontend |
| `--api-key` | `VIGIL_API_KEY` | (empty) | API key for auth. Empty = auth disabled |

#### Logging & Diagnostics

| Env Var | Default | Description |
|---------|---------|-------------|
| `VIGIL_LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `TZ` | `UTC` | IANA timezone for capture timestamps and directory names (e.g. `America/Chicago`). Defaults to UTC. |

#### Camera Tuning

| Env Var | Default | Description |
|---------|---------|-------------|
| `VIGIL_FSWEBCAM_DEVICE` | (auto) | Video device path, e.g. `/dev/video0` |
| `VIGIL_FSWEBCAM_RES` | (auto) | Capture resolution, e.g. `1280x960` |
| `VIGIL_FSWEBCAM_DELAY` | `2` | Seconds to wait for camera auto-exposure before capturing |
| `VIGIL_FSWEBCAM_ARGS` | (empty) | Extra flags passed to fswebcam |
| `VIGIL_CAPTURE_RETRIES` | `3` | Max capture attempts before giving up (handles "device busy") |

### Camera Drivers

- **mock** - Generates a colored test image with a timestamp. Good for development and verifying the pipeline works.
- **ffmpeg** - Captures a frame from a webcam or video device using FFmpeg. Works on Linux, macOS, and Windows. This is probably what you want for a USB webcam.
- **libcamera** - Uses libcamera-still for Raspberry Pi camera modules.
- **fswebcam** - Uses fswebcam for basic USB cameras on Linux. Recommended for Raspberry Pi 3B+ - more reliable than FFmpeg on USB 2.0 buses. Configure resolution with `VIGIL_FSWEBCAM_RES`.

### Authentication

If you set VIGIL_API_KEY, all API endpoints (except /api/status and /api/auth/*) require an Authorization: Bearer <key> header. The web UI will show a login screen and store the key in your browser local storage.

If you do not set an API key, auth is completely disabled and everything is open. This is fine for local use behind your own network, but if you expose Vigil to the internet you should set a key.

## Architecture

```
vigil/
  main.go                 # Entry point, wires everything together
  internal/
    api/                   # HTTP handlers and middleware (chi router)
    camera/                # Camera driver abstraction
    capture/               # Capture pipeline (camera -> disk -> DB -> hook)
    db/                    # SQLite database layer
    hook/                  # Post-capture hook runner
    logger/                # Structured leveled logging
    scheduler/             # Cron scheduler wrapper
  hooks/
    analyze.py             # Python image analysis hook (optional)
  web/
    src/                   # React + TypeScript frontend
```

The capture pipeline runs on each scheduled tick:
1. The scheduler fires based on cron expressions stored in SQLite
2. The camera driver takes a screenshot and writes it to data/captures/<date>/<time>.jpg
3. The capture gets logged in the database
4. If a hook script is configured, it runs with the image path as an argument and stdout is stored

### The Analysis Hook

The included hooks/analyze.py script calculates brightness, sharpness (Laplacian variance), file size, and compares each capture to the previous one to detect motion. Results come back as JSON and get displayed on the dashboard.

You do not have to use Python. Any executable that accepts an image path as its first argument and prints to stdout will work. The hook runner detects .py, .rb, and .js files and invokes the appropriate interpreter automatically.

To enable the hook, set the hook_path field on your schedule to hooks/analyze.py through the Config page.

#### Python Setup (for the analysis hook)

```bash
python -m venv .venv

# Windows
.\.venv\Scripts\activate

# Linux/macOS
source .venv/bin/activate

pip install -r hooks/requirements.txt
```

The Go binary will automatically look for .venv/Scripts/python.exe (Windows) or .venv/bin/python3 (Linux) before falling back to the system Python.

## Docker Details

The Dockerfile uses a multi-stage build:
1. Node stage builds the frontend with Vite
2. Go stage compiles the binary
3. Runtime stage uses python:3.12-slim, installs FFmpeg and the Python hook dependencies

This means the Docker image works out of the box with the analysis hook. No extra setup needed.

### Volumes

| Mount Point | Purpose | Required? |
|-------------|---------|-----------|
| `/data` | Captures (images), SQLite database, and any runtime state. This is where your images live. If you do not mount this, your captures will be lost when the container restarts. | Yes |
| `/home/vigil/hooks` | Hook scripts. The image ships with analyze.py baked in, but if you want to iterate on your own hooks without rebuilding the image, mount a local directory here. | No |

### Running with Docker

```bash
docker build -t vigil .
docker run -d \
  -p 8080:8080 \
  -v ./data:/data \
  -e VIGIL_API_KEY=secret \
  vigil
```

If you want to develop hooks outside the container:

```bash
docker run -d \
  -p 8080:8080 \
  -v ./data:/data \
  -v ./hooks:/home/vigil/hooks \
  -e VIGIL_API_KEY=secret \
  vigil
```

### Docker Compose

The included docker-compose.yml is a minimal starting point:

```yaml
services:
  vigil:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
      # Uncomment to develop hooks without rebuilding:
      # - ./hooks:/home/vigil/hooks
    environment:
      - VIGIL_CAMERA=mock
      # - VIGIL_API_KEY=changeme
    restart: unless-stopped
```

## Troubleshooting

### SQLite "Out of Memory" or Permission Denied

If you see `open database: ... out of memory (14)` in your Docker logs, it usually means the `/data` directory inside the container (mounted from your host) is not writable by the `vigil` user (UID 1000).

To fix this, run this on your host machine:
```bash
sudo chown -R 1000:1000 ./data
```

### Camera Device Access (Black Frames)

If you get black frames or your camera indicator doesn't turn on (common on Raspberry Pi 3B+):

1.  **Use fswebcam**: It's more robust for older USB buses. Set `VIGIL_CAMERA=fswebcam`.
2.  **Set a resolution**: Without `VIGIL_FSWEBCAM_RES`, fswebcam defaults to 352x288. Run `v4l2-ctl --list-formats-ext -d /dev/video0` on the Pi to see what your camera supports, then set e.g. `VIGIL_FSWEBCAM_RES=1280x960`.
3.  **Pass Devices**: You must pass the device and add the video group in `docker-compose.yml`:
    ```yaml
    devices:
      - "/dev/video0:/dev/video0"
    group_add:
      - video
    ```
4.  **Check Power**: Adding `max_usb_current=1` to your `/boot/config.txt` and rebooting can help if the camera resets under load.
5.  **"Device or resource busy"**: The Pi 3B+ USB subsystem can hold the camera device after a capture. Vigil retries automatically (configurable with `VIGIL_CAPTURE_RETRIES`, default 3). Failed captures are not logged to the database.
6.  **Enable debug logging**: Set `VIGIL_LOG_LEVEL=DEBUG` to see exactly what fswebcam/ffmpeg negotiates with the camera.

## Go Notes

If you are new to Go, here are some things worth knowing about this codebase:

- **internal/ convention** - Packages under internal/ cannot be imported by other Go modules. This is enforced by the compiler, not just a convention. It keeps the public API surface zero.
- **CGO_ENABLED=0** - The binary is built without cgo, which means it is fully static and can run on any Linux system without shared libraries. This is why we use modernc.org/sqlite instead of mattn/go-sqlite3.
- **Chi router** - We use go-chi/chi for routing. It is a lightweight router that plays well with the standard net/http interfaces.
- **No frameworks** - There is no web framework. Handlers are plain http.HandlerFuncs. Middleware is just function wrapping.

## Contributing

This project is in early stages. If you are interested in contributing, open an issue first so we can talk about it. PRs are welcome for bug fixes.

## License

MIT. See [LICENSE](./LICENSE).
