# 🌐 Video Cleaner Frontend

This is the React-based user interface for the Video Cleaner & Processor. It provides a user-friendly dashboard to configure video processing tasks and monitor their progress in real-time.

## ✨ Features
- **Dynamic Chapter Selector**: Interactive interface to select skip ranges based on scanned chapters.
- **Audio Track Picker**: Easily select which audio stream to prioritize.
- **Progress Tracking**: Real-time polling of the backend status providing live feedback.
- **Responsive Design**: Built with Tailwind CSS 4.0 for a modern look and feel.

## 🛠️ Tech Stack
- **React 19**
- **Vite**
- **TypeScript**
- **Tailwind CSS 4.0**
- **pnpm** (Package Manager)

## 🚀 Getting Started

### Prerequisites
- Node.js (v18+)
- pnpm installed (`npm i -g pnpm`)

### Installation
```bash
pnpm install
```

### Development
```bash
pnpm dev
```
The app will run at `http://localhost:5173`.

### Build
```bash
pnpm build
```

## 🔗 API Integration
The frontend communicates with the Go backend (default `http://localhost:8080`) through several endpoints:
- `GET /api/scan`: Fetches chapter and audio data.
- `POST /api/process`: Submits processing options.
- `GET /api/status`: Polls for real-time progress.

---
*For full project documentation, see the [main README](../README.md).*
