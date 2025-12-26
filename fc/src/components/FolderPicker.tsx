
interface Props {
    inputPath: string;
    setInputPath: (v: string) => void;
    outputPath: string;
    setOutputPath: (v: string) => void;
}

export default function FolderPicker({ inputPath, setInputPath, outputPath, setOutputPath }: Props) {
    return (
        <div className="flex gap-4 mb-4">
            <input
                className="border p-2 flex-1"
                placeholder="Input folder path"
                value={inputPath}
                onChange={e => setInputPath(e.target.value)}
            />
            <input
                className="border p-2 flex-1"
                placeholder="Output folder path"
                value={outputPath}
                onChange={e => setOutputPath(e.target.value)}
            />
        </div>
    );
}
