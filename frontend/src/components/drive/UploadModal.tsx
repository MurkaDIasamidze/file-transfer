import { useCallback, useState } from 'react';
import { UploadService } from '../../services/uploadService';
import type { UploadProgress } from '../../services/uploadService';

interface Props {
  folderId:   number | null;
  onClose:    () => void;
  onProgress: (p: UploadProgress) => void;
  onComplete: () => void;
}

interface QueueItem {
  file:     File;
  progress: UploadProgress | null;
  error:    string | null;
  done:     boolean;
}

export default function UploadModal({ folderId, onClose, onProgress, onComplete }: Props) {
  const [queue,    setQueue]    = useState<QueueItem[]>([]);
  const [dragging, setDragging] = useState(false);
  const [running,  setRunning]  = useState(false);

  const addFiles = (files: FileList | null) => {
    if (!files) return;
    setQueue(prev => [
      ...prev,
      ...Array.from(files).map(file => ({ file, progress: null, error: null, done: false })),
    ]);
  };

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    addFiles(e.dataTransfer.files);
  }, []);

  const handleStart = async () => {
    setRunning(true);
    for (let i = 0; i < queue.length; i++) {
      const item = queue[i];
      if (item.done) continue;

      const svc = new UploadService(
        (p: UploadProgress) => {
          onProgress(p);
          setQueue(prev => {
            const next = [...prev];
            next[i] = { ...next[i], progress: p };
            return next;
          });
        },
        () => {
          setQueue(prev => {
            const next = [...prev];
            next[i] = { ...next[i], done: true };
            return next;
          });
          onComplete();
        },
        (msg) => {
          setQueue(prev => {
            const next = [...prev];
            next[i] = { ...next[i], error: msg };
            return next;
          });
        },
      );

      try {
        await svc.upload(item.file, folderId);
      } catch { /* errors handled by svc */ }
    }
    setRunning(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl shadow-xl w-full max-w-md">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <h2 className="text-base font-semibold text-gray-900">Upload files</h2>
          <button onClick={onClose} className="w-7 h-7 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors">
            <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* Drop zone */}
          <div
            onDragOver={e => { e.preventDefault(); setDragging(true); }}
            onDragLeave={() => setDragging(false)}
            onDrop={handleDrop}
            className={`border-2 border-dashed rounded-xl p-8 text-center transition-colors cursor-pointer ${
              dragging ? 'border-blue-400 bg-blue-50' : 'border-gray-300 hover:border-gray-400'
            }`}
            onClick={() => document.getElementById('upload-input')?.click()}
          >
            <input
              id="upload-input"
              type="file"
              multiple
              className="hidden"
              onChange={e => addFiles(e.target.files)}
            />
            <svg className="w-10 h-10 mx-auto text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
            </svg>
            <p className="text-sm text-gray-500">
              <span className="text-blue-600 font-medium">Click to browse</span> or drag & drop
            </p>
            <p className="text-xs text-gray-400 mt-1">Any file type supported</p>
          </div>

          {/* Queue */}
          {queue.length > 0 && (
            <ul className="space-y-2 max-h-48 overflow-y-auto">
              {queue.map((item, i) => (
                <li key={i} className="flex items-center gap-3 text-sm">
                  <span className="text-lg">{item.done ? '✅' : item.error ? '❌' : '📄'}</span>
                  <div className="flex-1 min-w-0">
                    <p className="truncate text-gray-700">{item.file.name}</p>
                    {item.progress && !item.done && (
                      <div className="mt-1 h-1 bg-gray-100 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-blue-500 transition-all"
                          style={{ width: `${item.progress.percentage}%` }}
                        />
                      </div>
                    )}
                    {item.error && <p className="text-xs text-red-500">{item.error}</p>}
                  </div>
                  <span className="text-xs text-gray-400 shrink-0">
                    {item.done ? 'Done' : item.progress ? `${item.progress.percentage}%` : 'Waiting'}
                  </span>
                </li>
              ))}
            </ul>
          )}

          {/* Actions */}
          <div className="flex gap-3 pt-2">
            <button
              onClick={onClose}
              className="flex-1 py-2.5 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleStart}
              disabled={queue.length === 0 || running}
              className="flex-1 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white rounded-lg text-sm font-medium transition-colors"
            >
              {running ? 'Uploading…' : `Upload ${queue.length > 0 ? `(${queue.length})` : ''}`}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}