import { useState } from 'react';
import type { FileItem, Folder } from '../../types';

interface Props {
  file: FileItem;
  folders: Folder[];
  viewMode: 'my-drive' | 'recent' | 'starred' | 'trash';
  onMove: (folderId: number | null) => void;
  onToggleStar: () => void;
  onTrash: () => void;
  onRestore: () => void;
  onDelete: () => void;
}

const EXT_ICONS: Record<string, string> = {
  pdf: '📄',
  doc: '📝',
  docx: '📝',
  xls: '📊',
  xlsx: '📊',
  ppt: '📑',
  pptx: '📑',
  zip: '🗜️',
  rar: '🗜️',
  mp4: '🎬',
  mov: '🎬',
  avi: '🎬',
  mp3: '🎵',
  wav: '🎵',
  jpg: '🖼️',
  jpeg: '🖼️',
  png: '🖼️',
  gif: '🖼️',
  webp: '🖼️',
  txt: '📃',
  md: '📃',
};

function fileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() ?? '';
  return EXT_ICONS[ext] ?? '📁';
}

function fmtSize(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1024 ** 2) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 ** 3) return `${(b / 1024 ** 2).toFixed(1)} MB`;
  return `${(b / 1024 ** 3).toFixed(1)} GB`;
}

export default function FileCard({ file, folders, viewMode, onMove, onToggleStar, onTrash, onRestore, onDelete }: Props) {
  const [menu, setMenu] = useState(false);
  const [dragging, setDragging] = useState(false);

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('fileId', String(file.id));
    setDragging(true);
  };

  const handleDragEnd = () => {
    setDragging(false);
  };

  return (
    <div
      draggable={viewMode === 'my-drive'}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      className={`group relative flex flex-col items-center p-3 rounded-xl hover:bg-gray-100 cursor-pointer transition-all select-none ${
        dragging ? 'opacity-50' : ''
      }`}
    >
      {/* Icon */}
      <div className="relative">
        <div className="w-14 h-14 flex items-center justify-center bg-white rounded-xl border border-gray-200 shadow-sm text-3xl group-hover:shadow-md transition-shadow">
          {fileIcon(file.file_name)}
        </div>

        {/* Star badge */}
        {file.starred && (
          <div className="absolute -top-1 -right-1 w-5 h-5 bg-yellow-400 rounded-full flex items-center justify-center">
            <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
          </div>
        )}

        {/* Kebab */}
        <button
          className="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-gray-200 hover:bg-gray-300 hidden group-hover:flex items-center justify-center"
          onClick={(e) => {
            e.stopPropagation();
            setMenu((m) => !m);
          }}
        >
          <svg className="w-3 h-3 text-gray-600" fill="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="5" r="1.5" />
            <circle cx="12" cy="12" r="1.5" />
            <circle cx="12" cy="19" r="1.5" />
          </svg>
        </button>
      </div>

      <p className="mt-1 text-xs text-gray-700 text-center truncate w-full">{file.file_name}</p>
      <p className="text-xs text-gray-400">{fmtSize(file.file_size)}</p>

      {menu && (
        <div
          className="absolute top-10 right-0 z-20 bg-white rounded-lg shadow-lg border border-gray-200 py-1 w-40"
          onMouseLeave={() => setMenu(false)}
        >
          {viewMode === 'trash' ? (
            <>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onRestore();
                  setMenu(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
              >
                Restore
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                  setMenu(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
              >
                Delete Forever
              </button>
            </>
          ) : (
            <>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onToggleStar();
                  setMenu(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
              >
                {file.starred ? '★ Unstar' : '☆ Star'}
              </button>
              {folders.length > 0 && (
                <div className="border-t border-gray-100 my-1">
                  <p className="px-3 py-1 text-xs text-gray-400">Move to folder</p>
                  {folders.map((f) => (
                    <button
                      key={f.id}
                      onClick={(e) => {
                        e.stopPropagation();
                        onMove(f.id);
                        setMenu(false);
                      }}
                      className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
                    >
                      📁 {f.name}
                    </button>
                  ))}
                </div>
              )}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onTrash();
                  setMenu(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
              >
                Move to Trash
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}