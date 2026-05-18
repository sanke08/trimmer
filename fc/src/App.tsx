import { useEffect, useState } from "react";
import TrimSelector from "./components/TrimSelector";
import ChapterAdder from "./components/ChapterAdder";
import ChapterShifter from "./components/ChapterShifter";
import { scanFolder, submitTrimOptions, type AudioTrack, type TrimOptions, type Progress, type EpisodeStatus, type PartStatus } from "./api";
import FolderPicker from "./components/FolderPicker";

const STATE_ICON: Record<string, string> = {
  pending: "⏳",
  processing: "⚙️",
  merging: "🔗",
  done: "✅",
  failed: "❌",
};

const STATE_COLOR: Record<string, string> = {
  pending: "text-gray-400",
  processing: "text-blue-500",
  merging: "text-purple-500",
  done: "text-green-600",
  failed: "text-red-500",
};

function EpisodeRow({ ep }: { ep: EpisodeStatus }) {
  return (
    <div className="flex items-start gap-2 py-1 text-sm">
      <span className="shrink-0">{STATE_ICON[ep.state] ?? "•"}</span>
      <span className={`flex-1 truncate ${STATE_COLOR[ep.state] ?? ""}`}>{ep.name}</span>
      {ep.error && <span className="text-red-400 text-xs ml-2 break-all">{ep.error}</span>}
    </div>
  );
}

function PartRow({ part }: { part: PartStatus }) {
  return (
    <div className="flex items-start gap-2 py-1 text-sm">
      <span className="shrink-0">{STATE_ICON[part.state] ?? "•"}</span>
      <span className={`flex-1 ${STATE_COLOR[part.state] ?? ""}`}>{part.name}</span>
      {part.error && <span className="text-red-400 text-xs ml-2">{part.error}</span>}
    </div>
  );
}

function ProgressPanel({ progress }: { progress: Progress }) {
  const isProcessing = progress.status === "processing";
  const isMerging = progress.status === "merging";
  const isDone = progress.status === "done";

  const phaseDone = isProcessing
    ? `${progress.completed}/${progress.total} episodes trimmed`
    : isMerging
    ? `${progress.completed}/${progress.total} parts merged`
    : isDone
    ? "All done!"
    : progress.status;

  const barColor = isDone ? "bg-green-500" : isMerging ? "bg-purple-500" : "bg-blue-500";

  return (
    <div className="mb-6 border rounded-lg p-4 bg-gray-50">
      <div className="flex items-center justify-between mb-1">
        <span className="font-semibold capitalize">
          {isDone ? "✅ Done" : isProcessing ? "⚙️ Processing episodes" : isMerging ? "🔗 Merging parts" : progress.status}
        </span>
        <span className="text-sm text-gray-500">{Math.round(progress.percent)}%</span>
      </div>

      {/* Progress bar */}
      <div className="w-full bg-gray-200 rounded-full h-3 mb-2">
        <div
          className={`${barColor} h-3 rounded-full transition-all duration-500`}
          style={{ width: `${progress.percent}%` }}
        />
      </div>

      <p className="text-sm text-gray-600 mb-3">{phaseDone}</p>

      {/* Episode list */}
      {progress.episodes && progress.episodes.length > 0 && (
        <details open={!isDone} className="mb-2">
          <summary className="cursor-pointer text-sm font-medium text-gray-700 mb-1 select-none">
            Episodes ({progress.episodes.filter(e => e.state === "done").length}/{progress.episodes.length} done,{" "}
            {progress.episodes.filter(e => e.state === "failed").length} failed)
          </summary>
          <div className="mt-1 max-h-56 overflow-y-auto pl-1">
            {progress.episodes.map((ep, i) => (
              <EpisodeRow key={i} ep={ep} />
            ))}
          </div>
        </details>
      )}

      {/* Parts list */}
      {progress.parts && progress.parts.length > 0 && (
        <details open className="mb-2">
          <summary className="cursor-pointer text-sm font-medium text-gray-700 mb-1 select-none">
            Output Parts ({progress.parts.filter(p => p.state === "done").length}/{progress.parts.length} done,{" "}
            {progress.parts.filter(p => p.state === "failed").length} failed)
          </summary>
          <div className="mt-1 pl-1">
            {progress.parts.map((part, i) => (
              <PartRow key={i} part={part} />
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

export default function App() {
  const [inputPath, setInputPath] = useState("");
  const [outputPath, setOutputPath] = useState("");
  const [chapters, setChapters] = useState<string[]>([]);
  const [progress, setProgress] = useState<Progress | null>(null);
  const [audioTracks, setAudioTracks] = useState<AudioTrack[]>([]);
  const [selectedTrack, setSelectedTrack] = useState<number>(0);

  const [pollTick, setPollTick] = useState(0);
  const [mode, setMode] = useState<"trim" | "addChapter" | "shiftChapter">("trim");

  const [options, setOptions] = useState<TrimOptions>({
    skipRanges: [], // multiple ranges
    parts: 1,
    audioIndex: 0,
  });

  const [loading, setLoading] = useState(false);

  const handleScan = async () => {
    setLoading(true);
    try {
      const result = await scanFolder(inputPath.replaceAll("\\\\", "/"));
      setChapters(Object.keys(result.chapters));
      setAudioTracks(result.audioTracks);
    } catch (e) {
      console.error(e);
      alert("Error scanning folder");
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    setProgress(null);
    setLoading(true);
    try {
      await submitTrimOptions(inputPath, outputPath.replaceAll("\\\\", "/"), {
        ...options,
        audioIndex: selectedTrack,
      });
      setPollTick(t => t + 1);
    } catch (e) {
      console.error(e);
      alert("Error starting trim process");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (pollTick === 0) return;
    let interval: number;

    const poll = async () => {
      const res = await fetch("http://localhost:9000/api/status");
      const data = await res.json();
      setProgress(data);

      if (data.done) clearInterval(interval);
    };

    poll();
    interval = setInterval(poll, 2000);

    return () => clearInterval(interval);
  }, [pollTick]);

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-4">🎬 Anime Cleaner - Trim Selector</h1>

      <div className="flex gap-2 mb-4">
        <button
          className={
            mode === "trim"
              ? "bg-blue-600 text-white px-4 py-2"
              : "border bg-white text-gray-700 px-4 py-2"
          }
          onClick={() => setMode("trim")}
        >
          🎬 Trim
        </button>
        <button
          className={
            mode === "addChapter"
              ? "bg-blue-600 text-white px-4 py-2"
              : "border bg-white text-gray-700 px-4 py-2"
          }
          onClick={() => setMode("addChapter")}
        >
          🔖 Add Chapter
        </button>
        <button
          className={
            mode === "shiftChapter"
              ? "bg-blue-600 text-white px-4 py-2"
              : "border bg-white text-gray-700 px-4 py-2"
          }
          onClick={() => setMode("shiftChapter")}
        >
          🔀 Shift Chapter
        </button>
      </div>

      <FolderPicker
        inputPath={inputPath}
        setInputPath={setInputPath}
        outputPath={outputPath}
        setOutputPath={setOutputPath}
      />

      {mode === "trim" && (
        <>
          <div className="flex gap-4 mb-4">
            <button className="bg-blue-500 text-white px-4 py-2" onClick={handleScan}>
              Scan First Episode
            </button>
            <button className="bg-purple-500 text-white px-4 py-2" onClick={handleSubmit}>
              Submit Trim Options
            </button>
          </div>

          {loading && <p>Loading...</p>}

          {audioTracks.length > 0 && (
            <div className="mb-6">
              <h2 className="font-semibold mb-2">🎧 Select Audio Track:</h2>
              <select
                className="border p-2 rounded w-full"
                value={selectedTrack}
                onChange={(e) => setSelectedTrack(Number(e.target.value))}
              >
                {audioTracks.map((track) => (
                  <option key={track.index} value={track.index}>
                    #{track.index} — {track.lang || "Unknown"} {track.title && `(${track.title})`}
                  </option>
                ))}
              </select>
            </div>
          )}
        </>
      )}

      {progress && progress.status !== "idle" && (
        <ProgressPanel progress={progress} />
      )}

      {mode === "trim" && chapters.length > 0 && (
        <TrimSelector chapters={chapters} options={options} setOptions={setOptions} />
      )}

      {mode === "addChapter" && (
        <ChapterAdder
          inputPath={inputPath}
          outputPath={outputPath}
          onSubmitted={() => setPollTick(t => t + 1)}
        />
      )}

      {mode === "shiftChapter" && (
        <ChapterShifter
          inputPath={inputPath}
          outputPath={outputPath}
          onSubmitted={() => setPollTick(t => t + 1)}
        />
      )}
    </div>
  );
}
