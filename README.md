# 🎬 Video Cleaner & Processor

A professional, high-performance toolkit for batch processing video episodes (specifically MKV). This project combines a **Go** backend leveraging **FFmpeg** for lightning-fast video manipulation and a modern **React 19** frontend for an intuitive, real-time dashboard.

## 🚀 Key Features

- **Automated Chapter Scanning**: Quickly scans MKV files to detect internal chapters, enabling easy identification of Intros, Outros, and Recaps.
- **Precision Trimming**: Define multiple "Skip Ranges" (e.g., skip from *Opening* to *Episode Start*) to remove unwanted segments with frame accuracy.
- **Batched Parallel Processing**: Process an entire season of episodes simultaneously using Go's lightweight concurrency (goroutines).
- **Smart Episode Merging**: Combine your processed episodes into a user-defined number of "Parts" (e.g., merge 12 episodes into 3 large movie-like parts).
- **Metadata Preservation**: Automatically shifts and reapplies chapter markers and metadata to ensure your final files remain organized and navigatable.
- **Real-time Progress Dashboard**: Watch your processing queue in real-time with detailed status updates, completion percentages, and phase tracking (Scanning -> Processing -> Merging).
- **Automated Cleanup**: Intelligently removes temporary files and intermediate segments to keep your workspace clean.

---

## 🛠️ Tech Stack

### Backend (**BC**)
- **Language**: Go (Golang)
- **Engine**: FFmpeg & FFprobe (Command-line integration)
- **Architecture**: Modular Service-Oriented Design
- **API**: Standard `net/http` with CORS support

### Frontend (**fc**)
- **Framework**: React 19 + Vite
- **Styling**: Tailwind CSS 4.0
- **Type Safety**: TypeScript
- **State Management**: React Hooks (useState/useEffect) with polling for progress updates.

---

## 📋 Prerequisites

Before you begin, ensure you have the following installed:
- [Go](https://golang.org/dl/) (v1.21+)
- [Node.js](https://nodejs.org/) (v18+) & [pnpm](https://pnpm.io/)
- [FFmpeg](https://ffmpeg.org/download.html) (Ensure `ffmpeg` and `ffprobe` are in your system's PATH)

---

## ⚙️ Setup & Installation

### 1. Clone the Repository
```bash
git clone <repository-url>
cd "Video cleaner"
```

### 2. Backend Setup
```bash
cd BC
go mod download
```

### 3. Frontend Setup
```bash
cd fc
pnpm install
```

---

## 🏃 How to Run

To get the application up and running, you need to start both the backend and frontend servers.

### Step 1: Start the Backend
Open a terminal and run:
```bash
cd BC
go run main.go
```
*The server will start at `http://localhost:8080`*

### Step 2: Start the Frontend
Open another terminal and run:
```bash
cd fc
pnpm dev
```
*The UI will be accessible at `http://localhost:5173`*

---

## 📖 How it Works

1.  **Scan**: Enter your input folder path (containing `.mkv` files) and an output folder path. Click **Scan**.
2.  **Configure**:
    - The tool will analyze the first episode's chapters.
    - Select which segments to **skip** (e.g., Select "Opening" to "Episode Start" to skip the intro).
    - Select your preferred **Audio Track**.
    - Choose how many **Parts** to merge the output into.
3.  **Process**: Click **Submit**. The backend will:
    - Trim each episode based on your skip ranges.
    - Concat segments back into cleaned episodes.
    - Merge episodes into the requested number of parts.
    - Apply corrected chapter metadata to the final files.
4.  **Monitor**: Watch the progress bar until it hits **100% (Done)**.

---

## 📁 Project Structure

```text
Video cleaner/
├── BC/                 # Backend (Go)
│   ├── ffmpeg/         # FFmpeg command wrappers
│   ├── handlers/       # HTTP Route Handlers
│   ├── services/       # Core Business Logic (Processing, Merging)
│   ├── models/         # Data Structures & Types
│   └── main.go         # Entry Point
├── fc/                 # Frontend (React + Vite)
│   ├── src/
│   │   ├── components/ # Reusable UI Components
│   │   ├── api/        # Backend Communication Layer
│   │   └── App.tsx     # Main Application Logic
│   └── index.html      # Main HTML
└── README.md           # This file
```

## 🤝 Contributing
Feel free to open issues or submit pull requests to improve the tool!

## 📜 License
[MIT](LICENSE) (Change as per your preference)
