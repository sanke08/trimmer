import { useState } from "react";
import { addChapter } from "../api";

interface Props {
  inputPath: string;
  outputPath: string;
  onSubmitted: () => void;
}

/**
 * Parse a timestamp string into seconds.
 * Accepts:
 *   - "SS" or "SS.ms"        (e.g. "90", "90.5")
 *   - "MM:SS" or "MM:SS.ms"  (e.g. "1:30", "1:30.5")
 *   - "HH:MM:SS" or "HH:MM:SS.ms" (e.g. "00:01:30")
 * Returns null on invalid input. Empty string returns null.
 */
export function parseTimestamp(input: string): number | null {
  const trimmed = input.trim();
  if (trimmed === "") return null;

  const parts = trimmed.split(":");
  if (parts.length < 1 || parts.length > 3) return null;

  // Non-last parts must be non-negative integers.
  for (let i = 0; i < parts.length - 1; i++) {
    const p = parts[i];
    if (!/^\d+$/.test(p)) return null;
  }

  // Last part is a non-negative number that may have decimals.
  const lastStr = parts[parts.length - 1];
  if (!/^\d+(\.\d+)?$/.test(lastStr)) return null;

  const nums = parts.map((p) => Number(p));
  if (nums.some((n) => !Number.isFinite(n) || n < 0)) return null;

  if (parts.length === 1) {
    // SS(.ms) - no upper bound on raw seconds form
    return nums[0];
  }

  if (parts.length === 2) {
    // MM:SS - MM in [0,60), SS in [0,60)
    const [mm, ss] = nums;
    if (mm >= 60 || ss >= 60) return null;
    return mm * 60 + ss;
  }

  // HH:MM:SS - MM and SS in [0,60)
  const [hh, mm, ss] = nums;
  if (mm >= 60 || ss >= 60) return null;
  return hh * 3600 + mm * 60 + ss;
}

export default function ChapterAdder({ inputPath, outputPath, onSubmitted }: Props) {
  const [chapterName, setChapterName] = useState("");
  const [timestamp, setTimestamp] = useState("");
  const [loading, setLoading] = useState(false);
  const [lastResponse, setLastResponse] = useState<string>("");

  const parsed = parseTimestamp(timestamp);
  const timestampInvalid = timestamp.trim() !== "" && parsed === null;

  const canSubmit =
    !loading &&
    inputPath.trim() !== "" &&
    chapterName.trim() !== "" &&
    parsed !== null &&
    parsed >= 0;

  const handleSubmit = async () => {
    if (inputPath.trim() === "") {
      alert("Input folder path is required");
      return;
    }
    if (chapterName.trim() === "") {
      alert("Chapter name is required");
      return;
    }
    if (parsed === null || parsed < 0) {
      alert("Invalid timestamp");
      return;
    }

    setLoading(true);
    setLastResponse("");
    try {
      const resp = await addChapter({
        input: inputPath.replaceAll("\\", "/"),
        output: outputPath.replaceAll("\\", "/"),
        chapterName: chapterName.trim(),
        timeSeconds: parsed,
      });
      setLastResponse(JSON.stringify(resp));
      if (resp && resp.status === "no_files") {
        alert(`Backend says: ${resp.message}`);
        return;
      }
      onSubmitted();
    } catch (e) {
      console.error(e);
      setLastResponse(String(e));
      alert("Error starting add-chapter process — is the backend running on :9000?");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="border p-4 mb-4">
      <h2 className="font-bold mb-2">🔖 Add Chapter to All Episodes</h2>
      <p className="text-sm text-gray-500 mb-3">
        Adds the chapter to every .mkv in the input folder. If output is empty or same as input, files are updated in place.
      </p>

      <div className="mb-3">
        <label className="block text-sm font-medium mb-1">Chapter Name</label>
        <input
          type="text"
          className="border p-2 w-full"
          placeholder="Opening"
          value={chapterName}
          onChange={(e) => setChapterName(e.target.value)}
        />
      </div>

      <div className="mb-3">
        <label className="block text-sm font-medium mb-1">Timestamp</label>
        <div className="flex items-center gap-2">
          <input
            type="text"
            className="border p-2 flex-1"
            placeholder="00:01:30 or 90"
            value={timestamp}
            onChange={(e) => setTimestamp(e.target.value)}
          />
          {timestamp.trim() !== "" && (
            timestampInvalid ? (
              <span className="text-sm text-red-500">Invalid timestamp</span>
            ) : (
              <span className="text-sm text-gray-600">→ {parsed!.toFixed(2)}s</span>
            )
          )}
        </div>
        <p className="text-xs text-gray-500 mt-1">
          Accepts HH:MM:SS, MM:SS, or seconds (e.g., 90 or 1:30.5)
        </p>
      </div>

      <button
        className="bg-purple-500 text-white px-4 py-2 disabled:opacity-50 disabled:cursor-not-allowed"
        onClick={handleSubmit}
        disabled={!canSubmit}
      >
        Add Chapter to All Episodes
      </button>

      {loading && <p className="mt-2">Loading...</p>}
      {lastResponse && (
        <pre className="mt-2 text-xs bg-gray-100 p-2 rounded overflow-x-auto">
          Backend response: {lastResponse}
        </pre>
      )}
    </div>
  );
}
