import type { FileItem, Folder } from '../../types';
import FolderCard from './FolderCard';
import FileCard from './FileCard';

interface Props {
  folders:         Folder[];
  files:           FileItem[];
  viewMode:        'my-drive' | 'recent' | 'starred' | 'trash';
  onOpenFolder:    (f: Folder) => void;
  onTrashFolder:   (id: number) => void;
  onRestoreFolder: (id: number) => void;
  onDeleteFolder:  (id: number) => void;
  onMoveFile:      (fileId: number, folderId: number | null) => void;
  onToggleStar:    (id: number) => void;
  onTrashFile:     (id: number) => void;
  onRestoreFile:   (id: number) => void;
  onDeleteFile:    (id: number) => void;
}

export default function FileGrid({
  folders, files, viewMode,
  onOpenFolder, onTrashFolder, onRestoreFolder, onDeleteFolder,
  onMoveFile, onToggleStar, onTrashFile, onRestoreFile, onDeleteFile,
}: Props) {
  const empty = folders.length === 0 && files.length === 0;

  if (empty) {
    const emptyText: Record<string, string> = {
      'my-drive': 'This folder is empty',
      recent:     'No recent files',
      starred:    'No starred files',
      trash:      'Trash is empty',
    };
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-400">
        <svg className="w-16 h-16 mb-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1}
            d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <p className="text-lg font-medium">{emptyText[viewMode]}</p>
        <p className="text-sm mt-1">
          {viewMode === 'my-drive' ? 'Upload files or create folders to get started' : ''}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {folders.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-gray-500 mb-3">Folders</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {folders.map((f) => (
              <FolderCard
                key={f.id}
                folder={f}
                viewMode={viewMode}
                onOpen={() => onOpenFolder(f)}
                onTrash={() => onTrashFolder(f.id)}
                onRestore={() => onRestoreFolder(f.id)}
                onDelete={() => onDeleteFolder(f.id)}
                onDrop={(fileId) => onMoveFile(fileId, f.id)}
              />
            ))}
          </div>
        </section>
      )}

      {files.length > 0 && (
        <section>
          <h2 className="text-sm font-medium text-gray-500 mb-3">Files</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {files.map((f) => (
              <FileCard
                key={f.id}
                file={f}
                folders={folders}
                viewMode={viewMode}
                onMove={(folderId) => onMoveFile(f.id, folderId)}
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