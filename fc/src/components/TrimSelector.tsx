
// interface Props {
//     chapters: string[];
//     options: any;
//     setOptions: (opts: any) => void;
// }

import type { SkipRange } from "../api";

// export default function TrimSelector({ chapters, options, setOptions }: Props) {
//     return (
//         <div className="border p-4 mb-4">
//             <h2 className="font-bold mb-2">Select Trim Keys</h2>
//             <div className="flex gap-4 mb-2">
//                 <div>
//                     <label className="block">OP Start</label>
//                     <select
//                         className="border p-1"
//                         value={options.opStart}
//                         onChange={e => setOptions({ ...options, opStart: e.target.value })}
//                     >
//                         <option value="">--Select--</option>
//                         {chapters.map(k => <option key={k}>{k}</option>)}
//                     </select>
//                 </div>
//                 <div>
//                     <label className="block">OP End</label>
//                     <select
//                         className="border p-1"
//                         value={options.opEnd}
//                         onChange={e => setOptions({ ...options, opEnd: e.target.value })}
//                     >
//                         <option value="">--Select--</option>
//                         {chapters.map(k => <option key={k}>{k}</option>)}
//                     </select>
//                 </div>
//             </div>

//             <div className="flex gap-4 mb-2">
//                 <div>
//                     <label className="block">ED Start</label>
//                     <select
//                         className="border p-1"
//                         value={options.edStart}
//                         onChange={e => setOptions({ ...options, edStart: e.target.value })}
//                     >
//                         <option value="">--Select--</option>
//                         {chapters.map(k => <option key={k}>{k}</option>)}
//                     </select>
//                 </div>
//                 <div>
//                     <label className="block">ED End</label>
//                     <select
//                         className="border p-1"
//                         value={options.edEnd}
//                         onChange={e => setOptions({ ...options, edEnd: e.target.value })}
//                     >
//                         <option value="">--Select--</option>
//                         {chapters.map(k => <option key={k}>{k}</option>)}
//                     </select>
//                 </div>
//             </div>

//             <div>
//                 <label className="block">Number of Parts</label>
//                 <input
//                     type="number"
//                     min={1}
//                     className="border p-1 w-20"
//                     value={options.parts}
//                     onChange={e => setOptions({ ...options, parts: parseInt(e.target.value) })}
//                 />
//             </div>
//         </div>
//     );
// }














interface Props {
    chapters: string[];
    options: {
        skipRanges: SkipRange[];
        parts: number;
    };
    setOptions: (opts: any) => void;
}

export default function TrimSelector({ chapters, options, setOptions }: Props) {
    const addRange = () => {
        setOptions({ ...options, skipRanges: [...options.skipRanges, { start: "", end: "" }] });
    };

    const updateRange = (index: number, key: "start" | "end", value: string) => {
        const newRanges = [...options.skipRanges];
        newRanges[index][key] = value;
        setOptions({ ...options, skipRanges: newRanges });
    };

    const removeRange = (index: number) => {
        const newRanges = [...options.skipRanges];
        newRanges.splice(index, 1);
        setOptions({ ...options, skipRanges: newRanges });
    };

    return (
        <div className="border p-4 mb-4">
            <h2 className="font-bold mb-2">Select Trim Ranges (OP/ED or middle)</h2>

            {options.skipRanges.map((range, idx) => (
                <div key={idx} className="flex gap-2 mb-2 items-end">
                    <div>
                        <label>Start</label>
                        <select
                            className="border p-1"
                            value={range.start}
                            onChange={(e) => updateRange(idx, "start", e.target.value)}
                        >
                            <option value="">--Select--</option>
                            {chapters.map((c) => (
                                <option key={c}>{c}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label>End</label>
                        <select
                            className="border p-1"
                            value={range.end}
                            onChange={(e) => updateRange(idx, "end", e.target.value)}
                        >
                            <option value="">--Select--</option>
                            {chapters.map((c) => (
                                <option key={c}>{c}</option>
                            ))}
                        </select>
                    </div>

                    <button
                        className="bg-red-500 text-white px-2 py-1 rounded"
                        onClick={() => removeRange(idx)}
                    >
                        Remove
                    </button>
                </div>
            ))}

            <button
                className="bg-green-500 text-white px-4 py-2 rounded mb-2"
                onClick={addRange}
            >
                + Add Skip Range
            </button>

            <div>
                <label>Number of Parts</label>
                <input
                    type="number"
                    min={1}
                    className="border p-1 w-20"
                    value={options.parts}
                    onChange={(e) => setOptions({ ...options, parts: parseInt(e.target.value) })}
                />
            </div>
        </div>
    );
}
