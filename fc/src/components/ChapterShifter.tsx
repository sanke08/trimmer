import { useState } from "react";
import { shiftChapter } from "../api";

interface Props {
  inputPath: string;
  outputPath: string;
  onSubmitted: () => void;
}

export default function ChapterShifter({ inputPath, outputPath, onSubmitted }: Props) {
  const [name, setName] = useState("");
  const [sign, setSign] = useState<"+" | "-">("+");
  const [minutes, setMinutes] = useState("0");
  const [seconds, setSeconds] = useState("0");
  const [loading, setLoading] = useState(false);
  const [lastResponse, setLastResponse] = useState<string>("");

  const mNum = Number(minutes);
  const sNum = Number(seconds);
  const minutesValid = Number.isFinite(mNum) && mNum >= 0;
  const secondsValid = Number.isFinite(sNum) && sNum >= 0;
  const magnitude = (minutesValid ? mNum : 0) * 60 + (secondsValid ? sNum : 0);
  const delta = (sign === "+" ? 1 : -1) * magnitude;

  const canSubmit =
    !loading &&
    inputPath.trim() !== "" &&
    name.trim() !== "" &&
    minutesValid &&
    secondsValid &&
    magnitude > 0;

  const handleSubmit = async () => {
    if (inputPath.trim() === "") {
      alert("Input folder path is required");
      return;
    }
    if (name.trim() === "") {
      alert("Chapter name is required");
      return;
    }
    if (!minutesValid || !secondsValid || magnitude <= 0) {
      alert("Enter a non-zero shift amount (minutes and/or seconds)");
      return;
    }

    setLoading(true);
    setLastResponse("");
    try {
      const resp = await shiftChapter({
        input: inputPath.replaceAll("\\", "/"),
        output: outputPath.replaceAll("\\", "/"),
        chapterName: name.trim(),
        deltaSeconds: delta,
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
      alert("Error starting shift-chapter process — is the backend running on :9000?");
    } finally {
      setLoading(false);
    }
  };

  const dirBtn = (active: boolean) =>
    active
      ? "bg-blue-600 text-white px-4 py-2 text-lg font-semibold w-12"
      : "border bg-white text-gray-700 px-4 py-2 text-lg font-semibold w-12 hover:bg-gray-50";

  return (
    <div className="border p-4 mb-4">
      <h2 className="font-bold mb-2">🔀 Shift Existing Chapter</h2>
      <p className="text-sm text-gray-500 mb-3">
        Moves the START of the chapter (matched by name) forward or backward in every .mkv in the input folder.
        Other chapters renormalise around it. Out-of-range values are clamped.
      </p>

      <div className="mb-3">
        <label className="block text-sm font-medium mb-1">Chapter Name</label>
        <input
          type="text"
          className="border p-2 w-full"
          placeholder="Ep"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>

      <div className="mb-3">
        <label className="block text-sm font-medium mb-1">Direction</label>
        <div className="flex gap-2">
          <button
            type="button"
            className={dirBtn(sign === "+")}
            onClick={() => setSign("+")}
            title="Shift forward"
          >
            +
          </button>
          <button
            type="button"
            className={dirBtn(sign === "-")}
            onClick={() => setSign("-")}
            title="Shift backward"
          >
            −
          </button>
        </div>
      </div>

      <div className="mb-3 flex gap-3 items-end">
        <div className="flex-1">
          <label className="block text-sm font-medium mb-1">Minutes</label>
          <input
            type="number"
            min={0}
            step={1}
            className="border p-2 w-full"
            value={minutes}
            onChange={(e) => setMinutes(e.target.value)}
          />
        </div>
        <div className="flex-1">
          <label className="block text-sm font-medium mb-1">Seconds</label>
          <input
            type="number"
            min={0}
            step="any"
            className="border p-2 w-full"
            value={seconds}
            onChange={(e) => setSeconds(e.target.value)}
          />
        </div>
      </div>

      <p className="text-sm text-gray-600 mb-3">
        Shift amount: <span className="font-mono">{delta >= 0 ? "+" : ""}{delta.toFixed(2)}s</span>
      </p>

      <button
        className="bg-purple-500 text-white px-4 py-2 disabled:opacity-50 disabled:cursor-not-allowed"
        onClick={handleSubmit}
        disabled={!canSubmit}
      >
        Shift Chapter in All Episodes
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
