import { useState } from 'react';
import type { FileItem, Folder } from '../../types';

interface Props {
  file:         FileItem;
  folders:      Folder[];
  viewMode:     'my-drive' | 'recent' | 'starred' | 'trash';
  selected:     boolean;
  selecting:    boolean;
  onSelect:     (id: number) => void;
  onDownload:   () => void;
  onMove:       (folderId: number | null) => void;
  onToggleStar: () => void;
  onTrash:      () => void;
  onRestore:    () => void;
  onDelete:     () => void;
}

const EXT_ICONS: Record<string, string> = {
  pdf:'📄', doc:'📝', docx:'📝', xls:'📊', xlsx:'📊',
  ppt:'📑', pptx:'📑', zip:'🗜️', rar:'🗜️',
  mp4:'🎬', mov:'🎬', avi:'🎬', mp3:'🎵', wav:'🎵',
  jpg:'🖼️', jpeg:'🖼️', png:'🖼️', gif:'🖼️', webp:'🖼️',
  txt:'📃', md:'📃',
};
function fileIcon(name: string) {
  return EXT_ICONS[name.split('.').pop()?.toLowerCase() ?? ''] ?? '📁';
}
function fmtSize(b: number) {
  if (b < 1024) return `${b} B`;
  if (b < 1024**2) return `${(b/1024).toFixed(1)} KB`;
  if (b < 1024**3) return `${(b/1024**2).toFixed(1)} MB`;
  return `${(b/1024**3).toFixed(1)} GB`;
}

export default function FileCard({
  file, folders, viewMode,
  selected, selecting,
  onSelect, onDownload, onMove, onToggleStar, onTrash, onRestore, onDelete,
}: Props) {
  const [menu, setMenu]         = useState(false);
  const [dragging, setDragging] = useState(false);
  const [downloading, setDownloading] = useState(false);

  // Single click = select (when in selection mode), double click = download
  const handleClick = () => {
    if (selecting) { onSelect(file.id); return; }
  };

  const handleDoubleClick = () => {
    if (selecting) return;
    if (viewMode !== 'trash') triggerDownload();
  };

  const handleCheckbox = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSelect(file.id);
  };

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('fileId', String(file.id));
    setDragging(true);
  };

  const triggerDownload = async () => {
    if (downloading) return;
    setDownloading(true);
    try {
      onDownload();
    } finally {
      // Reset spinner after a short delay so the user sees feedback
      setTimeout(() => setDownloading(false), 1500);
    }
  };

  const isProcessing = file.status === 'processing';

  return (
    <div
      draggable={viewMode === 'my-drive' && !selecting}
      onDragStart={handleDragStart}
      onDragEnd={() => setDragging(false)}
      onClick={handleClick}
      onDoubleClick={handleDoubleClick}
      title={viewMode !== 'trash' && !selecting ? 'Double-click to download' : undefined}
      className={`group relative flex flex-col items-center p-3 rounded-xl transition-all select-none
        ${selecting ? 'cursor-pointer' : 'cursor-default'}
        ${selected  ? 'bg-blue-50 ring-2 ring-blue-400' : 'hover:bg-gray-100'}
        ${dragging  ? 'opacity-50' : ''}
      `}
    >
      {/* Checkbox */}
      <div
        onClick={handleCheckbox}
        className={`absolute top-1.5 left-1.5 z-10 w-5 h-5 rounded flex items-center justify-center transition-all cursor-pointer
          ${selected || selecting ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}
        `}
      >
        <div className={`w-4 h-4 rounded border-2 flex items-center justify-center transition-colors
          ${selected ? 'bg-blue-500 border-blue-500' : 'bg-white border-gray-400 hover:border-blue-400'}
        `}>
          {selected && (
            <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7"/>
            </svg>
          )}
        </div>
      </div>

      {/* Icon */}
      <div className="relative mt-1">
        <div className={`w-14 h-14 flex items-center justify-center bg-white rounded-xl border text-3xl shadow-sm transition-shadow
          ${selected ? 'border-blue-200 shadow-blue-100' : 'border-gray-200 group-hover:shadow-md'}
        `}>
          {isProcessing
            ? <span className="text-2xl animate-pulse">⏳</span>
            : fileIcon(file.file_name)
          }
        </div>

        {/* Star badge */}
        {file.starred && !isProcessing && (
          <div className="absolute -top-1 -right-1 w-5 h-5 bg-yellow-400 rounded-full flex items-center justify-center">
            <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
            </svg>
          </div>
        )}

        {/* Download spinner overlay */}
        {downloading && (
          <div className="absolute inset-0 flex items-center justify-center bg-white/70 rounded-xl">
            <svg className="w-5 h-5 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
            </svg>
          </div>
        )}

        {/* Kebab — hidden in selection mode */}
        {!selecting && (
          <button
            className="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-gray-200 hover:bg-gray-300 hidden group-hover:flex items-center justify-center"
            onClick={e => { e.stopPropagation(); setMenu(m => !m); }}
          >
            <svg className="w-3 h-3 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
              <circle cx="12" cy="5"  r="1.5"/>
              <circle cx="12" cy="12" r="1.5"/>
              <circle cx="12" cy="19" r="1.5"/>
            </svg>
          </button>
        )}
      </div>

      <p className="mt-1 text-xs text-gray-700 text-center truncate w-full">{file.file_name}</p>
      <p className="text-xs text-gray-400">
        {isProcessing ? 'Uploading to cloud…' : fmtSize(file.file_size)}
      </p>

      {/* Context menu */}
      {menu && !selecting && (
        <div
          className="absolute top-10 right-0 z-20 bg-white rounded-lg shadow-lg border border-gray-200 py-1 w-48"
          onMouseLeave={() => setMenu(false)}
        >
          {viewMode === 'trash' ? (
            <>
              <button onClick={e => { e.stopPropagation(); onRestore(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/>
                </svg>
                Restore
              </button>
              <button onClick={e => { e.stopPropagation(); onDelete(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                Delete Forever
              </button>
            </>
          ) : (
            <>
              {/* Download — disabled while still processing */}
              <button
                disabled={isProcessing}
                onClick={e => { e.stopPropagation(); triggerDownload(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/>
                </svg>
                {isProcessing ? 'Uploading…' : 'Download'}
              </button>

              <div className="border-t border-gray-100 my-1"/>

              <button onClick={e => { e.stopPropagation(); onToggleStar(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                <svg className="w-3.5 h-3.5" fill={file.starred ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"/>
                </svg>
                {file.starred ? 'Unstar' : 'Star'}
              </button>

              {folders.length > 0 && (
                <>
                  <div className="border-t border-gray-100 my-1"/>
                  <p className="px-3 py-1 text-xs text-gray-400">Move to folder</p>
                  {folders.map(f => (
                    <button key={f.id}
                      onClick={e => { e.stopPropagation(); onMove(f.id); setMenu(false); }}
                      className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                      📁 {f.name}
                    </button>
                  ))}
                </>
              )}

              <div className="border-t border-gray-100 my-1"/>
              <button onClick={e => { e.stopPropagation(); onTrash(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                Move to Trash
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}