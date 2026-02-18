import { useEffect, useState, useCallback, useMemo } from 'react';
import { filesApi, foldersApi } from '../../services/api';
import type { FileItem, Folder, BreadcrumbItem } from '../../types';
import type { UploadProgress } from '../../services/uploadService';
import Header from './Header';
import Sidebar from './Sidebar';
import FileGrid from './FileGrid';
import UploadModal from './UploadModal';
import CreateFolderModal from './CreateFolderModal';

type ViewMode = 'my-drive' | 'recent' | 'starred' | 'trash';

export default function DrivePage() {
  const [folders,     setFolders]     = useState<Folder[]>([]);
  const [files,       setFiles]       = useState<FileItem[]>([]);
  const [currentId,   setCurrentId]   = useState<number | null>(null);
  const [viewMode,    setViewMode]    = useState<ViewMode>('my-drive');
  const [breadcrumbs, setBreadcrumbs] = useState<BreadcrumbItem[]>([{ id: null, name: 'My Drive' }]);
  const [loading,     setLoading]     = useState(false);
  const [showUpload,  setShowUpload]  = useState(false);
  const [showFolder,  setShowFolder]  = useState(false);
  const [uploads,     setUploads]     = useState<Map<number, UploadProgress>>(new Map());
  const [search,      setSearch]      = useState('');

  const load = useCallback(async (folderId: number | null, mode: ViewMode) => {
    setLoading(true);
    try {
      if (mode === 'recent') {
        const { data } = await filesApi.recent();
        setFiles(data);
        setFolders([]);
      } else if (mode === 'starred') {
        const { data } = await filesApi.starred();
        setFiles(data);
        setFolders([]);
      } else if (mode === 'trash') {
        const [fRes, fiRes] = await Promise.all([
          foldersApi.listTrashed(),
          filesApi.trash(),
        ]);
        setFolders(fRes.data);
        setFiles(fiRes.data);
      } else {
        const [fRes, fiRes] = await Promise.all([
          foldersApi.list(folderId),
          filesApi.list(folderId),
        ]);
        setFolders(fRes.data);
        setFiles(fiRes.data);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(currentId, viewMode);
  }, [currentId, viewMode, load]);

  const filteredFolders = useMemo(() => {
    if (!search.trim()) return folders;
    return folders.filter((f) => f.name.toLowerCase().includes(search.toLowerCase()));
  }, [folders, search]);

  const filteredFiles = useMemo(() => {
    if (!search.trim()) return files;
    return files.filter((f) => f.file_name.toLowerCase().includes(search.toLowerCase()));
  }, [files, search]);

  const openFolder = (f: Folder) => {
    setViewMode('my-drive');
    setCurrentId(f.id);
    setSearch('');
    setBreadcrumbs((prev) => [...prev, { id: f.id, name: f.name }]);
  };

  const goToBreadcrumb = (item: BreadcrumbItem) => {
    setViewMode('my-drive');
    setCurrentId(item.id);
    setSearch('');
    const idx = breadcrumbs.findIndex((b) => b.id === item.id);
    setBreadcrumbs(breadcrumbs.slice(0, idx + 1));
  };

  const changeView = (mode: ViewMode) => {
    setViewMode(mode);
    setSearch('');
    setCurrentId(null);
    setBreadcrumbs([{
      id: null,
      name: mode === 'recent' ? 'Recent' : mode === 'starred' ? 'Starred' : mode === 'trash' ? 'Trash' : 'My Drive',
    }]);
  };

  const handleTrashFolder = async (id: number) => {
    await foldersApi.trash(id);
    load(currentId, viewMode);
  };

  const handleRestoreFolder = async (id: number) => {
    await foldersApi.restore(id);
    load(currentId, viewMode);
  };

  const handleDeleteFolder = async (id: number) => {
    await foldersApi.delete(id);
    load(currentId, viewMode);
  };

  const handleMoveFile = async (fileId: number, folderId: number | null) => {
    await filesApi.move(fileId, folderId);
    load(currentId, viewMode);
  };

  const handleToggleStar = async (id: number) => {
    await filesApi.toggleStar(id);
    load(currentId, viewMode);
  };

  const handleTrashFile = async (id: number) => {
    await filesApi.moveToTrash(id);
    load(currentId, viewMode);
  };

  const handleRestoreFile = async (id: number) => {
    await filesApi.restore(id);
    load(currentId, viewMode);
  };

  const handleDeleteFile = async (id: number) => {
    await filesApi.delete(id);
    load(currentId, viewMode);
  };

  const onUploadProgress = (p: UploadProgress) => {
    setUploads((prev) => new Map(prev).set(p.fileUploadId, p));
  };

  const onUploadComplete = () => {
    setTimeout(() => load(currentId, viewMode), 500);
  };

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      <Sidebar
        activeView={viewMode}
        onViewChange={changeView}
        onNewFolder={() => setShowFolder(true)}
        onNewFile={() => setShowUpload(true)}
      />

      <div className="flex flex-col flex-1 overflow-hidden">
        <Header onSearch={setSearch} />

        <main className="flex-1 overflow-auto p-6">
          {viewMode === 'my-drive' && (
            <nav className="flex items-center gap-1 text-sm mb-6 text-gray-500">
              {breadcrumbs.map((b, i) => (
                <span key={b.id ?? 'root'} className="flex items-center gap-1">
                  {i > 0 && <span className="text-gray-300">/</span>}
                  <button
                    onClick={() => goToBreadcrumb(b)}
                    className={`hover:text-blue-600 transition-colors ${
                      i === breadcrumbs.length - 1 ? 'text-gray-900 font-medium' : ''
                    }`}
                  >
                    {b.name}
                  </button>
                </span>
              ))}
            </nav>
          )}

          {search && (
            <p className="text-sm text-gray-500 mb-4">
              Results for <span className="font-medium text-gray-800">"{search}"</span>
            </p>
          )}

          {viewMode === 'my-drive' && !search && (
            <div className="flex items-center gap-3 mb-6">
              <button
                onClick={() => setShowUpload(true)}
                className="flex items-center gap-2 px-4 py-2 bg-white border border-gray-200 rounded-full text-sm font-medium text-gray-700 hover:bg-gray-50 shadow-sm transition-colors"
              >
                <svg className="w-4 h-4 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                </svg>
                New
              </button>
              <button
                onClick={() => setShowFolder(true)}
                className="flex items-center gap-2 px-4 py-2 bg-white border border-gray-200 rounded-full text-sm font-medium text-gray-700 hover:bg-gray-50 shadow-sm transition-colors"
              >
                <svg className="w-4 h-4 text-yellow-500" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M10 4H4c-1.11 0-2 .89-2 2v12c0 1.1.89 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
                </svg>
                New Folder
              </button>
            </div>
          )}

          {uploads.size > 0 && (
            <div className="mb-6 space-y-2">
              {[...uploads.values()].map((up) => (
                <div key={up.fileUploadId} className="bg-white rounded-lg border border-gray-200 p-3 flex items-center gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-700 truncate">{up.fileName}</p>
                    <div className="mt-1 h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-blue-500 transition-all duration-300 rounded-full"
                        style={{ width: `${up.percentage}%` }}
                      />
                    </div>
                  </div>
                  <span className="text-xs text-gray-500 shrink-0">{up.percentage}%</span>
                  {up.status === 'completed' && (
                    <svg className="w-4 h-4 text-green-500 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                  )}
                </div>
              ))}
            </div>
          )}

          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full animate-spin" />
            </div>
          ) : (
            <FileGrid
              folders={filteredFolders}
              files={filteredFiles}
              viewMode={viewMode}
              onOpenFolder={openFolder}
              onTrashFolder={handleTrashFolder}
              onRestoreFolder={handleRestoreFolder}
              onDeleteFolder={handleDeleteFolder}
              onMoveFile={handleMoveFile}
              onToggleStar={handleToggleStar}
              onTrashFile={handleTrashFile}
              onRestoreFile={handleRestoreFile}
              onDeleteFile={handleDeleteFile}
            />
          )}
        </main>
      </div>

      {showUpload && (
        <UploadModal
          folderId={currentId}
          onClose={() => setShowUpload(false)}
          onProgress={onUploadProgress}
          onComplete={onUploadComplete}
        />
      )}

      {showFolder && (
        <CreateFolderModal
          parentId={currentId}
          onClose={() => setShowFolder(false)}
          onCreated={() => load(currentId, viewMode)}
        />
      )}
    </div>
  );
}