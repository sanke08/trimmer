// import { useEffect, useState } from "react";
// import TrimSelector from "./components/TrimSelector";
// import { scanFolder, submitTrimOptions, type AudioTrack, type TrimOptions } from "./api";
// import FolderPicker from "./components/FolderPicker";

// export default function App() {
//   const [inputPath, setInputPath] = useState("");
//   const [outputPath, setOutputPath] = useState("");
//   const [chapters, setChapters] = useState<string[]>([]);
//   const [progress, setProgress] = useState<{ [key: string]: number }>()
//   const [audioTracks, setAudioTracks] = useState<AudioTrack[]>([]); // 👈 new
//   const [selectedTrack, setSelectedTrack] = useState<number>(0); // 👈 new


//   const [isPooling, setIsPooling] = useState(false)
//   const [options, setOptions] = useState<TrimOptions>({
//     opStart: "",
//     opEnd: "",
//     edStart: "",
//     edEnd: "",
//     parts: 1
//   });
//   const [loading, setLoading] = useState(false);

//   const handleScan = async () => {
//     setLoading(true);
//     try {
//       const result = await scanFolder(inputPath.replaceAll("\\\\", "/"));
//       setChapters(Object.keys(result.chapters));
//       console.log(result)
//       setAudioTracks(result.audioTracks);
//     } catch (e) {
//       console.error(e);
//       alert("Error scanning folder");
//     } finally {
//       setLoading(false);
//     }
//   };

//   const handleSubmit = async () => {
//     setIsPooling(false);
//     setLoading(true);
//     try {
//       await submitTrimOptions(inputPath, outputPath.replaceAll("\\\\", "/"), {
//         ...options,
//         audioIndex: selectedTrack, // 👈 include user’s audio choice
//       });
//       setIsPooling(true);
//     } catch (e) {
//       console.error(e);
//       alert("Error starting trim process");
//     } finally {
//       setLoading(false);
//     }
//   };


//   useEffect(() => {
//     if (!isPooling) return
//     let interval: number;

//     const poll = async () => {
//       const res = await fetch("http://localhost:8080/api/status");
//       const data = await res.json();
//       setProgress(data);

//       // ✅ stop polling if done
//       if (data.done) {
//         clearInterval(interval);
//       }
//     };

//     poll();
//     interval = setInterval(poll, 2000);

//     return () => clearInterval(interval);
//   }, [isPooling]);



//   return (
//     <div className="p-6 max-w-3xl mx-auto">
//       <h1 className="text-2xl font-bold mb-4">🎬 Anime Cleaner - Trim Selector</h1>
//       <FolderPicker
//         inputPath={inputPath}
//         setInputPath={setInputPath}
//         outputPath={outputPath}
//         setOutputPath={setOutputPath}
//       />
//       <div className="flex gap-4 mb-4">
//         <button className="bg-blue-500 text-white px-4 py-2" onClick={handleScan}>Scan First Episode</button>
//         <button className="bg-purple-500 text-white px-4 py-2" onClick={handleSubmit}>Submit Trim Options</button>
//       </div>

//       {loading && <p>Loading...</p>}



//       {
//         progress &&
//         <div>
//           <p>Status: {progress.status}</p>
//           <p>{progress.completed}/{progress.total} ({progress?.percent}%)</p>
//         </div>
//       }

//       {audioTracks.length > 0 && (
//         <div className="mb-6">
//           <h2 className="font-semibold mb-2">🎧 Select Audio Track:</h2>
//           <select
//             className="border p-2 rounded w-full"
//             value={selectedTrack}
//             onChange={(e) => setSelectedTrack(Number(e.target.value))}
//           >
//             {audioTracks.map((track) => (
//               <option key={track.index} value={track.index}>
//                 #{track.index} — {track.lang || "Unknown"} {track.title && `(${track.title})`}
//               </option>
//             ))}
//           </select>
//         </div>
//       )}

//       {progress && (
//         <div>
//           <p>Status: {progress.status}</p>
//           <p>
//             {progress.completed}/{progress.total} ({progress?.percent}%)
//           </p>
//         </div>
//       )}


//       {chapters.length > 0 && <TrimSelector chapters={chapters} options={options} setOptions={setOptions} />}
//     </div>
//   );
// }















import { useEffect, useState } from "react";
import TrimSelector from "./components/TrimSelector";
import { scanFolder, submitTrimOptions, type AudioTrack, type TrimOptions } from "./api";
import FolderPicker from "./components/FolderPicker";

export default function App() {
  const [inputPath, setInputPath] = useState("");
  const [outputPath, setOutputPath] = useState("");
  const [chapters, setChapters] = useState<string[]>([]);
  const [progress, setProgress] = useState<any>();
  const [audioTracks, setAudioTracks] = useState<AudioTrack[]>([]);
  const [selectedTrack, setSelectedTrack] = useState<number>(0);

  const [isPooling, setIsPooling] = useState(false);

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
    setIsPooling(false);
    setLoading(true);
    try {
      await submitTrimOptions(inputPath, outputPath.replaceAll("\\\\", "/"), {
        ...options,
        audioIndex: selectedTrack,
      });
      setIsPooling(true);
    } catch (e) {
      console.error(e);
      alert("Error starting trim process");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!isPooling) return;
    let interval: number;

    const poll = async () => {
      const res = await fetch("http://localhost:8080/api/status");
      const data = await res.json();
      setProgress(data);

      if (data.done) clearInterval(interval);
    };

    poll();
    interval = setInterval(poll, 2000);

    return () => clearInterval(interval);
  }, [isPooling]);

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-4">🎬 Anime Cleaner - Trim Selector</h1>

      <FolderPicker
        inputPath={inputPath}
        setInputPath={setInputPath}
        outputPath={outputPath}
        setOutputPath={setOutputPath}
      />

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

      {progress && (
        <div>
          <p>Status: {progress.status}</p>
          <p>
            {progress.completed}/{progress.total} ({progress?.percent}%)
          </p>
        </div>
      )}

      {chapters.length > 0 && (
        <TrimSelector chapters={chapters} options={options} setOptions={setOptions} />
      )}
    </div>
  );
}
