import { useState } from 'react';
import type { Folder } from '../../types';

interface Props {
  folder:    Folder;
  viewMode:  'my-drive' | 'recent' | 'starred' | 'trash';
  selected:  boolean;
  selecting: boolean;
  onSelect:  (id: number) => void;
  onOpen:    () => void;
  onTrash:   () => void;
  onRestore: () => void;
  onDelete:  () => void;
  onDrop:    (fileId: number) => void;
}

export default function FolderCard({
  folder, viewMode,
  selected, selecting,
  onSelect, onOpen, onTrash, onRestore, onDelete, onDrop,
}: Props) {
  const [menu,     setMenu]     = useState(false);
  const [dragOver, setDragOver] = useState(false);

  const handleClick = () => {
    if (selecting) {
      onSelect(folder.id);
    }
  };

  const handleCheckbox = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSelect(folder.id);
  };

  const handleDragOver = (e: React.DragEvent) => {
    if (selecting) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOver(true);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const fileId = e.dataTransfer.getData('fileId');
    if (fileId) onDrop(parseInt(fileId, 10));
  };

  return (
    <div
      onClick={handleClick}
      onDoubleClick={viewMode !== 'trash' && !selecting ? onOpen : undefined}
      onDragOver={handleDragOver}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
      className={`group relative flex flex-col items-center p-3 rounded-xl transition-all select-none
        ${selecting ? 'cursor-pointer' : 'cursor-default'}
        ${selected
          ? 'bg-blue-50 ring-2 ring-blue-400'
          : dragOver
          ? 'bg-blue-50 ring-2 ring-blue-400'
          : 'hover:bg-gray-100'}
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
          ${selected
            ? 'bg-blue-500 border-blue-500'
            : 'bg-white border-gray-400 hover:border-blue-400'}
        `}>
          {selected && (
            <svg className="w-2.5 h-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7"/>
            </svg>
          )}
        </div>
      </div>

      {/* Folder icon */}
      <div className="relative mt-1">
        <svg
          className={`w-14 h-14 transition-colors ${
            dragOver ? 'text-blue-400' : selected ? 'text-blue-400' : 'text-gray-400 group-hover:text-blue-400'
          }`}
          viewBox="0 0 24 24" fill="currentColor"
        >
          <path d="M10 4H4c-1.11 0-2 .89-2 2v12c0 1.1.89 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
        </svg>

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

      <p className="mt-1 text-xs text-gray-700 text-center truncate w-full">{folder.name}</p>

      {/* Drop hint overlay */}
      {dragOver && (
        <div className="absolute inset-0 flex items-center justify-center bg-blue-50/80 rounded-xl pointer-events-none">
          <span className="text-xs font-medium text-blue-600">Drop here</span>
        </div>
      )}

      {/* Context menu */}
      {menu && !selecting && (
        <div
          className="absolute top-10 right-0 z-20 bg-white rounded-lg shadow-lg border border-gray-200 py-1 w-40"
          onMouseLeave={() => setMenu(false)}
        >
          {viewMode === 'trash' ? (
            <>
              <button onClick={e => { e.stopPropagation(); onRestore(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                Restore
              </button>
              <button onClick={e => { e.stopPropagation(); onDelete(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50">
                Delete Forever
              </button>
            </>
          ) : (
            <>
              <button onClick={e => { e.stopPropagation(); onOpen(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                Open
              </button>
              <button onClick={e => { e.stopPropagation(); onTrash(); setMenu(false); }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-sm text-red-600 hover:bg-red-50">
                Move to Trash
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}