import { useCallback, useEffect, useRef, useState } from 'react';
import {
  uploadBatch,
  filesToTasks,
  collectEntriesSync,
  entriesToTasks,
} from '../../services/uploadService';
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
  if (b < 1024)          return `${b} B`;
  if (b < 1_048_576)     return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1_073_741_824) return `${(b / 1_048_576).toFixed(1)} MB`;
  return `${(b / 1_073_741_824).toFixed(1)} GB`;
}

export default function UploadModal({ folderId, onClose, onProgress, onComplete }: Props) {
  const token = useAuthStore(s => s.token)!;

  const [queue,    setQueue]    = useState<QueueItem[]>([]);
  const [dragging, setDragging] = useState(false);
  const [running,  setRunning]  = useState(false);
  const [mode,     setMode]     = useState<'files' | 'folder'>('files');

  const fileRef   = useRef<HTMLInputElement>(null);
  const folderRef = useRef<HTMLInputElement>(null);

  // Stable ref so the auto-close timer never needs onClose as a dependency
  // (onClose is a new arrow fn every DrivePage render, which would keep
  //  cancelling and restarting the timeout).
  const onCloseRef = useRef(onClose);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);

  // ── Add files to queue ─────────────────────────────────────────────────────

  const enqueue = useCallback((tasks: UploadTask[]) => {
    if (tasks.length === 0) return;
    setQueue(prev => [
      ...prev,
      ...tasks.map(task => ({ task, progress: null, error: null, done: false })),
    ]);
  }, []);

  // For <input> file / folder picker
  const handleInputChange = useCallback((files: FileList | null) => {
    if (!files) return;
    enqueue(filesToTasks(files));
  }, [enqueue]);

  // ── Drag & drop ────────────────────────────────────────────────────────────
  //
  // IMPORTANT: webkitGetAsEntry() must be called synchronously inside the event
  // handler before it returns — the DataTransfer object becomes invalid after.
  // We snapshot the FileSystemEntry objects first, then resolve them async.

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
    setDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    // Only clear if leaving the drop zone itself, not a child element
    if (e.currentTarget.contains(e.relatedTarget as Node)) return;
    setDragging(false);
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);

    // Snapshot entries SYNCHRONOUSLY before the event handler returns
    const entries = collectEntriesSync(e.dataTransfer);

    if (entries.length > 0) {
      // FileSystem API path — handles folders recursively
      try {
        const tasks = await entriesToTasks(entries);
        enqueue(tasks);
      } catch (err) {
        console.error('[drop] failed to read entries:', err);
        // Fall back to flat file list
        enqueue(filesToTasks(e.dataTransfer.files));
      }
    } else {
      // Browser doesn't support webkitGetAsEntry — fall back to flat file list
      enqueue(filesToTasks(e.dataTransfer.files));
    }
  }, [enqueue]);

  // ── Start upload ───────────────────────────────────────────────────────────

  const handleStart = async () => {
    if (queue.length === 0 || running) return;
    setRunning(true);

    const tasks = queue.map(q => q.task);

    await uploadBatch(
      tasks,
      folderId,
      token,
      (p: UploadProgress) => {
        onProgress(p);

        // Match by relPath (folder uploads) or fileName (plain files)
        const key = p.relPath || p.fileName;
        setQueue(prev => prev.map(item => {
          const itemKey = item.task.relPath || item.task.file.name;
          if (itemKey !== key) return item;
          return { ...item, progress: p, done: p.status === 'completed' };
        }));

        if (p.status === 'completed') onComplete();
      },
      (msg: string) => {
        // Mark the first non-done, non-errored item as failed
        setQueue(prev => {
          let marked = false;
          return prev.map(item => {
            if (!marked && !item.done && !item.error) {
              marked = true;
              return { ...item, error: msg };
            }
            return item;
          });
        });
      },
    );

    setRunning(false);
  };

  // ── Derived ────────────────────────────────────────────────────────────────

  const totalSize = queue.reduce((s, q) => s + q.task.file.size, 0);
  const doneCount = queue.filter(q => q.done).length;
  const allDone   = queue.length > 0 && doneCount === queue.length;

  // Auto-close 1.5 s after everything finishes
  useEffect(() => {
    if (!allDone) return;
    const t = setTimeout(() => onCloseRef.current(), 1500);
    return () => clearTimeout(t);
  }, [allDone]); // stable — onClose accessed via ref

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg">

        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-100">
          <h2 className="text-base font-semibold text-gray-900">Upload</h2>
          <button
            onClick={onClose}
            className="w-7 h-7 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors"
          >
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
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            onClick={() => (mode === 'folder' ? folderRef : fileRef).current?.click()}
            className={`border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer select-none ${
              dragging
                ? 'border-blue-400 bg-blue-50 scale-[1.01]'
                : 'border-gray-200 hover:border-blue-300 hover:bg-blue-50/30'
            }`}
          >
            <input
              ref={fileRef}
              type="file"
              multiple
              className="hidden"
              onChange={e => handleInputChange(e.target.files)}
            />
            <input
              ref={folderRef}
              type="file"
              // @ts-expect-error – webkitdirectory not in React's types
              webkitdirectory=""
              multiple
              className="hidden"
              onChange={e => handleInputChange(e.target.files)}
            />

            <div className="text-4xl mb-3">{dragging ? '📂' : mode === 'folder' ? '📁' : '📄'}</div>
            <p className="text-sm font-medium text-gray-700">
              {dragging
                ? 'Drop to add files…'
                : mode === 'folder'
                ? 'Click to select a folder, or drag & drop one here'
                : 'Click to browse files, or drag & drop here'}
            </p>
            <p className="text-xs text-gray-400 mt-1">
              Folders are supported via drag & drop
            </p>
          </div>

          {/* Queue list */}
          {queue.length > 0 && (
            <div className="space-y-1.5 max-h-52 overflow-y-auto">
              <div className="flex items-center justify-between text-xs text-gray-400 mb-2">
                <span>{queue.length} file{queue.length !== 1 ? 's' : ''}</span>
                <span>{fmtSize(totalSize)}</span>
              </div>

              {queue.map(item => {
                const key  = item.task.relPath || item.task.file.name;
                const pct  = item.progress?.percentage ?? 0;
                const icon = item.done ? '✅' : item.error ? '❌' : '📄';

                return (
                  <div key={key} className="flex items-center gap-2.5 bg-gray-50 rounded-lg px-3 py-2">
                    <span className="text-base shrink-0">{icon}</span>

                    <div className="flex-1 min-w-0">
                      <p className="truncate text-gray-700 text-xs font-medium">{key}</p>
                      {item.progress && !item.done && (
                        <div className="mt-1 h-1 bg-gray-200 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-blue-500 transition-all duration-200 rounded-full"
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                      )}
                      {item.error && <p className="text-xs text-red-500 mt-0.5">{item.error}</p>}
                    </div>

                    <span className="text-xs text-gray-400 shrink-0 w-12 text-right">
                      {item.done   ? 'Done'
                      : item.error ? 'Failed'
                      : item.progress ? `${pct}%`
                      : fmtSize(item.task.file.size)}
                    </span>
                  </div>
                );
              })}
            </div>
          )}

          {/* All-done banner */}
          {allDone && (
            <div className="flex items-center gap-2 text-sm text-green-700 bg-green-50 px-4 py-2.5 rounded-xl border border-green-100">
              <svg className="w-4 h-4 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
              </svg>
              All {queue.length} file{queue.length !== 1 ? 's' : ''} uploaded — closing…
            </div>
          )}

          {/* Action buttons */}
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
                {running
                  ? `Uploading… (${doneCount}/${queue.length})`
                  : `Upload${queue.length > 0 ? ` (${queue.length})` : ''}`
                }
              </button>
            )}
          </div>

        </div>
      </div>
    </div>
  );
}