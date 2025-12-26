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

// Scan first episode
export async function scanFolder(path: string): Promise<ScanResult> {
    const res = await fetch(`http://localhost:8080/api/scan?path=${encodeURIComponent(path)}`);
    return res.json();
}

// Submit trim options for all episodes
export async function submitTrimOptions(input: string, output: string, options: TrimOptions) {
    const res = await fetch("http://localhost:8080/api/process", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ input, output, options }),
    });
    return res.json();
}
