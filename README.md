# Trimmer

A video trimming and merging tool for batch-processing episode folders. Define skip ranges and part splits, then process entire seasons with parallel FFmpeg pipelines and monitor real-time progress from a browser UI.

---

## Overview

Trimmer takes a folder of video episodes, applies configurable skip ranges (e.g. intros, outros), and merges the remaining segments into N numbered output parts (`Part1.mkv`, `Part2.mkv`, etc.). A React frontend handles trim configuration, audio track selection, chapter manipulation, and progress display.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25.4, standard library only |
| Video processing | FFmpeg + ffprobe via `exec.Command` |
| Frontend | React 19, Vite 7.1.7, TypeScript |
| Styling | Tailwind CSS 4.1.14 |
| Backend port | 9000 |
| Frontend dev port | 5173 |

The frontend hardcodes `http://localhost:9000` as the backend base URL in `fc/src/api.ts`.

---

## Architecture

```
fc/src/
├── App.tsx              # Root state, polling loop, mode switching
├── api.ts               # All HTTP calls → http://localhost:9000
└── components/
    ├── FolderPicker     # Input/output path inputs
    ├── TrimSelector     # Skip ranges, parts count, audio track selector
    ├── ChapterAdder     # Add chapter at HH:MM:SS timestamp
    └── ChapterShifter   # Shift named chapter by ± seconds

BC/
├── handlers/            # One handler per endpoint
│   ├── scan.go
│   ├── process.go
│   ├── status.go
│   ├── add_chapter.go
│   └── shift_chapter.go
├── services/            # Business logic
│   ├── video_processor.go
│   ├── merger.go
│   ├── chapter_adder.go
│   └── chapter_shifter.go
├── ffmpeg/              # FFmpeg/ffprobe wrappers
│   ├── commands.go
│   ├── trimmer.go
│   └── metadata.go
├── middleware/cors.go
├── models/types.go
└── utils/
    ├── cleanup.go
    └── helpers.go
```

### Processing Pipeline

1. Episodes in the input folder are natural-sorted (Episode 2 before Episode 10).
2. `sync.WaitGroup` spawns one goroutine per episode.
3. Each goroutine applies the configured skip ranges via FFmpeg trim filters.
4. Trimmed segments are concatenated into N parts using FFmpeg concat demuxer.
5. Chapter metadata is applied to each output file.
6. Final files are written as `Part1.mkv`, `Part2.mkv`, etc.

### Progress Tracking

A global `ProgressState` struct is updated by the processing pipeline and read by the status handler. The frontend polls `GET /api/status` every 2 seconds. Status transitions: `processing` → `merging` → `done`.

---

## Features

- Batch-process entire episode folders in parallel goroutines
- Define multiple skip ranges per session, applied uniformly to every episode
- Split merged output into N numbered parts
- Select audio track by stream index
- Real-time per-episode and per-part progress via polling
- Add chapters to an existing video at a specific timestamp
- Shift existing chapter timestamps forward or backward by a delta in seconds

---

## API Endpoints

### `POST /api/scan?path={folder}`

Scans the first episode in a folder to discover chapter markers and audio tracks.

**Query param:** `path` — absolute path to the episode folder.

**Response:**
```json
{
  "chapters":    { "Opening": 93.5, "Part A": 185.0 },
  "audioTracks": [{ "index": 1, "lang": "jpn", "title": "Japanese" }],
  "firstFile":   "/media/episodes/Episode 01.mkv"
}
```

---

### `POST /api/process`

Starts the background batch-processing job. Returns `202 Accepted` immediately.

**Request body:**
```json
{
  "input":  "/media/episodes",
  "output": "/media/output",
  "options": {
    "skipRanges": [{ "Start": 90, "End": 150 }],
    "parts":      2,
    "audioIndex": 1
  }
}
```

---

### `GET /api/status`

Returns a snapshot of the current job's progress. Poll every 2 seconds.

**Response:**
```json
{
  "Total":     12,
  "Completed": 4,
  "Percent":   33.3,
  "Status":    "processing",
  "Episodes":  [{ "Name": "Episode 01.mkv", "State": "done", "Error": "" }],
  "Parts":     [{ "Name": "Part1.mkv", "State": "pending", "Error": "" }],
  "Done":      false
}
```

---

### `POST /api/add-chapter`

Injects a new chapter marker at a given timestamp into all MKV files in the output folder.

**Request body:**
```json
{
  "input":       "/media/episodes",
  "output":      "/media/output",
  "chapterName": "Opening",
  "timeSeconds": 120.5
}
```

---

### `POST /api/shift-chapter`

Shifts an existing chapter marker by a delta across all files in the output folder.

**Request body:**
```json
{
  "input":        "/media/episodes",
  "output":       "/media/output",
  "chapterName":  "Opening",
  "deltaSeconds": -5.0
}
```

Negative `deltaSeconds` shifts the chapter earlier; positive shifts it later.

---

## Components

| Component | Responsibility |
|---|---|
| `App.tsx` | Root state, 2-second polling loop, coordinates all child components |
| `FolderPicker` | Input/output folder path text inputs and Scan button |
| `TrimSelector` | Skip range list editor, parts count input, audio track dropdown |
| `ChapterAdder` | Chapter name + `HH:MM:SS` timestamp input, submits to `/api/add-chapter` |
| `ChapterShifter` | Chapter name, direction toggle (±), minutes + seconds inputs, submits to `/api/shift-chapter` |

---

## Setup & Run

### Prerequisites

- Go 1.25.4+
- Node.js 18+
- `ffmpeg` and `ffprobe` available on `$PATH`

No environment variables are required.

### Backend

```bash
cd BC
go run .
# Server starts on :9000
```

### Frontend

```bash
cd fc
npm install
npm run dev
# Dev server starts on :5173
```

Open `http://localhost:5173`. The backend must be running for API calls to succeed.

## Architecture

```
Browser (React 19 + Vite, port 5173)
    │
    ├──▶ FolderPicker — input/output folder paths
    ├──▶ TrimSelector — skip ranges, parts count, audio track
    ├──▶ ChapterAdder — add chapter at timestamp
    ├──▶ ChapterShifter — shift chapter by delta
    │
    └──▶ api.ts → HTTP calls to http://localhost:9000
                │
                ▼
    Go Backend (port 9000)
        │
        ├──▶ handlers/ — HTTP request parsing
        ├──▶ services/ — orchestration + goroutines
        │        video_processor.go — parallel episode processing
        │        merger.go — FFmpeg concat + metadata
        │        chapter_adder.go — add chapter to all episodes
        │        chapter_shifter.go — shift chapter timestamps
        │
        ├──▶ ffmpeg/ — exec.Command wrappers
        │        ffmpeg/ffprobe CLI calls
        │
        └──▶ Filesystem (input MKV files → processed output MKVs)
```

## User Flow

1. **Scan** → enter input folder path → FolderPicker → POST /api/scan?path=... → Go scans first episode with ffprobe → returns chapters {name:seconds} + audioTracks [{index,lang,title}] + firstFile name

2. **Configure** → TrimSelector: add skip ranges (start+end timestamps for OP/ED sections), set number of output parts, select audio track index

3. **Process** → click Process → POST /api/process {input, output, options:{skipRanges, parts, audioIndex}} → Go spawns goroutine → returns 202 immediately

4. **Monitor** → frontend polls GET /api/status every 2s → Progress{Total,Completed,Percent,Status,Episodes[],Parts[],Done} → progress bars update

5. **Done** → Status.Done=true → processed MKV files in output folder as Part1.mkv, Part2.mkv, etc.

6. **Add Chapter** → ChapterAdder: enter chapter name + timestamp (HH:MM:SS or seconds) → POST /api/add-chapter → adds chapter marker to all episodes

7. **Shift Chapter** → ChapterShifter: chapter name + direction(+/-) + delta seconds → POST /api/shift-chapter → shifts existing chapter timestamp across all episodes

## Data Flow

```
User clicks Process
    │
    ▼ POST /api/process { input, output, options }
Go handler (process.go)
    │ resets global ProgressState
    │ go services.ProcessEpisodes(input, output, options)
    │
    ▼ (returns 202 immediately)

services.ProcessEpisodes() [goroutine]
    │
    ├──▶ natural-sort episodes (Episode 2 before Episode 10)
    │
    └──▶ for each episode: go func() { ... } (sync.WaitGroup)
              │
              ├──▶ ffmpeg.ScanChapters(file) → chapters map
              ├──▶ ffmpeg.TrimEpisode(file, skipRanges) → trimmed temp files
              └──▶ ProgressState.Episodes[i].State = "done"
    │
    ▼ WaitGroup.Wait() — all episodes complete
    │
    └──▶ services.MergeEpisodes(tempFiles, parts, outputDir)
              │
              ├──▶ split into N parts
              ├──▶ for each part: write concat list → ffmpeg -f concat -c copy
              └──▶ apply chapter metadata → Part1.mkv, Part2.mkv, ...
    │
    ▼ ProgressState.Done = true

Frontend polls GET /api/status
    └──▶ reads global ProgressState → JSON response
```
