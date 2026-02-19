import { useCallback, useEffect, useRef, useState } from 'react';
import { uploadBatch, filesToTasks } from '../../services/uploadService';
import type { UploadProgress, UploadTask } from '../../services/uploadService';
import { useAuthStore } from '../../store/authStore';

interface Props {
  folderId:   number | null;
  onClose:    () => void;
  onProgress: (p: UploadProgress) => void;
  onComplete: () => void;
}

interface QueueItem {
  task:     UploadTask;
  progress: UploadProgress | null;
  error:    string | null;
  done:     boolean;
}

function fmtSize(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1048576) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1073741824) return `${(b / 1048576).toFixed(1)} MB`;
  return `${(b / 1073741824).toFixed(1)} GB`;
}

// FIX: match a queue item to a progress event by both relPath and fileName.
// For folder uploads relPath is e.g. "myfolder/report.pdf"; for single files
// relPath is '' and we fall back to fileName alone.
function matchesTask(item: QueueItem, p: UploadProgress): boolean {
  const taskKey  = item.task.relPath || item.task.file.name;
  const progKey  = p.relPath         || p.fileName;
  return taskKey === progKey;
}

export default function UploadModal({ folderId, onClose, onProgress, onComplete }: Props) {
  const token = useAuthStore(s => s.token)!;
  const [queue,    setQueue]    = useState<QueueItem[]>([]);
  const [dragging, setDragging] = useState(false);
  const [running,  setRunning]  = useState(false);
  const [mode,     setMode]     = useState<'files' | 'folder'>('files');
  const fileRef   = useRef<HTMLInputElement>(null);
  const folderRef = useRef<HTMLInputElement>(null);

  const addFiles = useCallback((files: FileList | null) => {
    if (!files) return;
    const tasks = filesToTasks(files);
    setQueue(prev => [
      ...prev,
      ...tasks.map(task => ({ task, progress: null, error: null, done: false })),
    ]);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    addFiles(e.dataTransfer.files);
  }, [addFiles]);

  const handleStart = async () => {
    if (queue.length === 0 || running) return;
    setRunning(true);

    const tasks = queue.map(q => q.task);

    try {
      await uploadBatch(
        tasks,
        folderId,
        token,
        (p) => {
          // Bubble up to DrivePage for the top progress strip
          onProgress(p);

          // FIX: use matchesTask so folder-upload items (relPath !== '') are found
          setQueue(prev => prev.map(item =>
            matchesTask(item, p)
              ? { ...item, progress: p, done: p.status === 'completed' }
              : item
          ));

          // Refresh the file list as soon as each individual file completes
          if (p.status === 'completed') {
            onComplete();
          }
        },
        (msg) => {
          setQueue(prev => prev.map(item =>
            !item.done && !item.error ? { ...item, error: msg } : item
          ));
        },
      );
    } catch (err) {
      console.error('Upload batch failed', err);
    } finally {
      setRunning(false);
    }
  };

  const totalSize = queue.reduce((sum, q) => sum + q.task.file.size, 0);
  const allDone   = queue.length > 0 && queue.every(q => q.done);

  // FIX: auto-close the modal 1.5 s after all files finish uploading
  useEffect(() => {
    if (!allDone) return;
    const t = setTimeout(() => onClose(), 1500);
    return () => clearTimeout(t);
  }, [allDone, onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-base font-semibold text-gray-900">Upload</h2>
          <button onClick={onClose} className="w-7 h-7 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors">
            <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* Mode tabs */}
          <div className="flex gap-1 p-1 bg-gray-100 rounded-xl">
            {(['files', 'folder'] as const).map(m => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={`flex-1 py-1.5 rounded-lg text-sm font-medium transition-all ${
                  mode === m ? 'bg-white shadow text-gray-900' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {m === 'files' ? '📄 Files' : '📁 Folder'}
              </button>
            ))}
          </div>

          {/* Drop zone */}
          <div
            onDragOver={e => { e.preventDefault(); setDragging(true); }}
            onDragLeave={() => setDragging(false)}
            onDrop={handleDrop}
            onClick={() => mode === 'folder' ? folderRef.current?.click() : fileRef.current?.click()}
            className={`border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer ${
              dragging ? 'border-blue-400 bg-blue-50 scale-[1.01]' : 'border-gray-200 hover:border-blue-300 hover:bg-blue-50/30'
            }`}
          >
            {/* Hidden inputs */}
            <input
              ref={fileRef}
              type="file"
              multiple
              className="hidden"
              onChange={e => addFiles(e.target.files)}
            />
            <input
              ref={folderRef}
              type="file"
              // @ts-expect-error webkitdirectory is not in React's HTMLInputElement types
              webkitdirectory=""
              multiple
              className="hidden"
              onChange={e => addFiles(e.target.files)}
            />

            <div className="text-4xl mb-3">{mode === 'folder' ? '📁' : '📄'}</div>
            <p className="text-sm font-medium text-gray-700">
              {mode === 'folder' ? 'Click to select a folder' : 'Click to browse files'}
            </p>
            <p className="text-xs text-gray-400 mt-1">or drag & drop here</p>
          </div>

          {/* Queue */}
          {queue.length > 0 && (
            <div className="space-y-1.5 max-h-52 overflow-y-auto">
              <div className="flex items-center justify-between text-xs text-gray-400 mb-2">
                <span>{queue.length} file{queue.length > 1 ? 's' : ''}</span>
                <span>{fmtSize(totalSize)}</span>
              </div>
              {queue.map((item) => {
                const displayName = item.task.relPath || item.task.file.name;
                return (
                  <div key={`${item.task.relPath || item.task.file.name}-${item.task.file.size}`} className="flex items-center gap-2.5 text-sm bg-gray-50 rounded-lg px-3 py-2">
                    <span className="text-base shrink-0">
                      {item.done ? '✅' : item.error ? '❌' : '📄'}
                    </span>
                    <div className="flex-1 min-w-0">
                      <p className="truncate text-gray-700 text-xs font-medium">{displayName}</p>
                      {item.progress && !item.done && (
                        <div className="mt-1 h-1 bg-gray-200 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-blue-500 transition-all duration-300 rounded-full"
                            style={{ width: `${item.progress.percentage}%` }}
                          />
                        </div>
                      )}
                      {item.error && <p className="text-xs text-red-500 mt-0.5">{item.error}</p>}
                    </div>
                    <span className="text-xs text-gray-400 shrink-0">
                      {item.done ? 'Done' : item.progress ? `${item.progress.percentage}%` : fmtSize(item.task.file.size)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}

          {/* All done feedback before auto-close */}
          {allDone && (
            <div className="flex items-center gap-2 text-sm text-green-700 bg-green-50 px-4 py-2.5 rounded-xl border border-green-100">
              <svg className="w-4 h-4 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
              </svg>
              All files uploaded! Closing…
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-3 pt-1">
            <button
              onClick={onClose}
              className="flex-1 py-2.5 border border-gray-200 rounded-xl text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              {allDone ? 'Close' : 'Cancel'}
            </button>
            {!allDone && (
              <button
                onClick={handleStart}
                disabled={queue.length === 0 || running}
                className="flex-1 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-300 text-white rounded-xl text-sm font-medium transition-colors flex items-center justify-center gap-2"
              >
                {running && (
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
                  </svg>
                )}
                {running ? 'Uploading…' : `Upload${queue.length > 0 ? ` (${queue.length})` : ''}`}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}