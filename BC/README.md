# ⚙️ Video Cleaner Backend

The backend engine for the Video Cleaner & Processor, written in Go. It handles the heavy lifting of video scanning, trimming, and merging by orchestrating FFmpeg commands.

## 🚀 Key Responsibilities
- **Path Scanning**: Locates MKV files and extracts technical metadata.
- **FFmpeg Orchestration**: Generates and executes complex FFmpeg commands for stream-copy trimming and merging.
- **Concurrency Control**: Manages parallel processing of multiple episodes to maximize CPU utilization.
- **Progress Management**: Maintains a thread-safe global state for real-time progress reporting.
- **Metadata Management**: Extracts, shifts, and reapplies FFmetadata to maintain chapter integrity across processed files.

## 📁 Architecture
The project follows a modular Go package structure:
- `/ffmpeg`: Low-level wrappers for `ffmpeg` and `ffprobe`.
- `/services`: High-level business logic (e.g., `ProcessEpisodes`, `MergeEpisodes`).
- `/handlers`: HTTP API endpoints.
- `/models`: Shared data structures and thread-safe state.
- `/utils`: Helper functions and cleanup logic.

## 🛠️ API Endpoints

### `GET /api/scan?path=<folder_path>`
Scans the specified folder for MKV files and returns the chapter list and audio tracks of the first episode.

### `POST /api/process`
Starts the video processing task.
**Body:**
```json
{
  "input": "string",
  "output": "string",
  "options": {
    "skipRanges": [ { "start": "Chapter1", "end": "Chapter2" } ],
    "parts": 12,
    "audioIndex": 0
  }
}
```

### `GET /api/status`
Returns the current processing status.
**Response:**
```json
{
  "total": 24,
  "completed": 5,
  "percent": 20.8,
  "status": "processing",
  "done": false
}
```

## 🏃 Running Locally

### Prerequisites
- Go 1.21+
- FFmpeg installed and in PATH.

### Commands
```bash
go mod download
go run main.go
```
The server will start on `http://localhost:8080`.

---
*For full project documentation, see the [main README](../README.md).*
