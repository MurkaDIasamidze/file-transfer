import { useEffect } from 'react';
import type { FileItem, Folder } from '../../types';
import FolderCard from './FolderCard';
import FileCard from './FileCard';

export type SelectionKey = `file-${number}` | `folder-${number}`;

interface Props {
  folders:         Folder[];
  files:           FileItem[];
  viewMode:        'my-drive' | 'recent' | 'starred' | 'trash';
  // selection
  selected:        Set<SelectionKey>;
  onSelect:        (key: SelectionKey) => void;
  onClearSelection:() => void;
  // individual actions
  onOpenFolder:    (f: Folder) => void;
  onTrashFolder:   (id: number) => void;
  onRestoreFolder: (id: number) => void;
  onDeleteFolder:  (id: number) => void;
  onMoveFile:      (fileId: number, folderId: number | null) => void;
  onToggleStar:    (id: number) => void;
  onTrashFile:     (id: number) => void;
  onRestoreFile:   (id: number) => void;
  onDeleteFile:    (id: number) => void;
  // bulk actions
  onBulkTrash:     () => void;
  onBulkRestore:   () => void;
  onBulkDelete:    () => void;
  onBulkStar:      () => void;
}

export default function FileGrid({
  folders, files, viewMode,
  selected, onSelect, onClearSelection,
  onOpenFolder, onTrashFolder, onRestoreFolder, onDeleteFolder,
  onMoveFile, onToggleStar, onTrashFile, onRestoreFile, onDeleteFile,
  onBulkTrash, onBulkRestore, onBulkDelete, onBulkStar,
}: Props) {
  const totalItems  = folders.length + files.length;
  const selectedCount = selected.size;
  const selecting   = selectedCount > 0;
  const allSelected = totalItems > 0 && selectedCount === totalItems;

  // Escape key clears selection
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && selecting) onClearSelection();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [selecting, onClearSelection]);

  const toggleSelectAll = () => {
    if (allSelected) {
      onClearSelection();
    } else {
      folders.forEach(f => onSelect(`folder-${f.id}`));
      files.forEach(f   => onSelect(`file-${f.id}`));
    }
  };

  const empty = totalItems === 0;

  if (empty) {
    const label: Record<string, string> = {
      'my-drive': 'This folder is empty',
      recent:     'No recent files',
      starred:    'No starred files',
      trash:      'Trash is empty',
    };
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-400">
        <svg className="w-16 h-16 mb-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1}
            d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
        </svg>
        <p className="text-lg font-medium">{label[viewMode]}</p>
        {viewMode === 'my-drive' && (
          <p className="text-sm mt-1">Upload files or create folders to get started</p>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">

      {/* ── Bulk action bar ─────────────────────────────────────────────────── */}
      {selecting && (
        <div className="flex items-center gap-2 px-4 py-2.5 bg-blue-600 rounded-xl shadow-lg text-white text-sm animate-in fade-in slide-in-from-top-2 duration-150">

          {/* Select-all checkbox */}
          <button
            onClick={toggleSelectAll}
            className="flex items-center gap-2 mr-1 shrink-0"
            title={allSelected ? 'Deselect all' : 'Select all'}
          >
            <div className={`w-4 h-4 rounded border-2 flex items-center justify-center transition-colors
              ${allSelected ? 'bg-white border-white' : 'border-white/70 hover:border-white'}
            `}>
              {allSelected && (
                <svg className="w-2.5 h-2.5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7"/>
                </svg>
              )}
              {!allSelected && selectedCount > 0 && (
                <div className="w-2 h-0.5 bg-white/70 rounded"/>
              )}
            </div>
          </button>

          <span className="font-medium shrink-0">
            {selectedCount} selected
          </span>

          <div className="flex-1"/>

          {/* Context-aware bulk actions */}
          {viewMode === 'trash' ? (
            <>
              <button
                onClick={onBulkRestore}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-white/20 hover:bg-white/30 rounded-lg transition-colors"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"/>
                </svg>
                Restore
              </button>
              <button
                onClick={onBulkDelete}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-red-500/80 hover:bg-red-500 rounded-lg transition-colors"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                Delete forever
              </button>
            </>
          ) : (
            <>
              <button
                onClick={onBulkStar}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-white/20 hover:bg-white/30 rounded-lg transition-colors"
              >
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                </svg>
                Star
              </button>
              <button
                onClick={onBulkTrash}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-red-500/80 hover:bg-red-500 rounded-lg transition-colors"
              >
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                Trash
              </button>
            </>
          )}

          {/* Clear / close */}
          <button
            onClick={onClearSelection}
            className="ml-1 p-1.5 bg-white/20 hover:bg-white/30 rounded-lg transition-colors"
            title="Clear selection (Esc)"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>
      )}

      {/* ── Folders ──────────────────────────────────────────────────────────── */}
      {folders.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-gray-500 mb-3">Folders</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {folders.map(f => (
              <FolderCard
                key={f.id}
                folder={f}
                viewMode={viewMode}
                selected={selected.has(`folder-${f.id}`)}
                selecting={selecting}
                onSelect={(id) => onSelect(`folder-${id}`)}
                onOpen={() => onOpenFolder(f)}
                onTrash={() => onTrashFolder(f.id)}
                onRestore={() => onRestoreFolder(f.id)}
                onDelete={() => onDeleteFolder(f.id)}
                onDrop={fileId => onMoveFile(fileId, f.id)}
              />
            ))}
          </div>
        </section>
      )}

      {/* ── Files ────────────────────────────────────────────────────────────── */}
      {files.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-gray-500 mb-3">Files</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {files.map(f => (
              <FileCard
                key={f.id}
                file={f}
                folders={folders}
                viewMode={viewMode}
                selected={selected.has(`file-${f.id}`)}
                selecting={selecting}
                onSelect={(id) => onSelect(`file-${id}`)}
                onMove={folderId => onMoveFile(f.id, folderId)}
                onToggleStar={() => onToggleStar(f.id)}
                onTrash={() => onTrashFile(f.id)}
                onRestore={() => onRestoreFile(f.id)}
                onDelete={() => onDeleteFile(f.id)}
              />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}