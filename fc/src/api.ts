export interface Chapters {
  [key: string]: number;
}

export interface AudioTrack {
  index: number;
  lang: string;
  title: string;
}

export interface ScanResult {
  chapters: Chapters;
  audioTracks: AudioTrack[];
  firstFile: string;
}

export interface SkipRange {
  start: string;
  end: string;
}

export interface TrimOptions {
  skipRanges: SkipRange[];
  parts: number;
  audioIndex?: number;
}

export interface EpisodeStatus {
  name: string;
  state: "pending" | "processing" | "done" | "failed";
  error?: string;
}

export interface PartStatus {
  name: string;
  state: "pending" | "merging" | "done" | "failed";
  error?: string;
}

export interface Progress {
  total: number;
  completed: number;
  percent: number;
  status: string;
  done: boolean;
  episodes?: EpisodeStatus[];
  parts?: PartStatus[];
}

// Scan first episode
export async function scanFolder(path: string): Promise<ScanResult> {
  const res = await fetch(
    `http://localhost:9000/api/scan?path=${encodeURIComponent(path)}`,
  );
  return res.json();
}

// Submit trim options for all episodes
export async function submitTrimOptions(
  input: string,
  output: string,
  options: TrimOptions,
) {
  const res = await fetch("http://localhost:9000/api/process", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ input, output, options }),
  });
  return res.json();
}

export interface AddChapterRequest {
  input: string;
  output: string;
  chapterName: string;
  timeSeconds: number;
}

export async function addChapter(req: AddChapterRequest) {
  const res = await fetch("http://localhost:9000/api/add-chapter", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return res.json();
}

export interface ShiftChapterRequest {
  input: string;
  output: string;
  chapterName: string;
  deltaSeconds: number;
}

export async function shiftChapter(req: ShiftChapterRequest) {
  const res = await fetch("http://localhost:9000/api/shift-chapter", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return res.json();
}
