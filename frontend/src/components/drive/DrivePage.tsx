import { useEffect, useState, useCallback, useMemo } from 'react';
import { filesApi, foldersApi } from '../../services/api';
import type { FileItem, Folder, BreadcrumbItem } from '../../types';
import type { UploadProgress } from '../../services/uploadService';
import type { SelectionKey } from './FileGrid';
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

  // ── Selection ──────────────────────────────────────────────────────────────
  // Key format: "file-{id}" | "folder-{id}"
  const [selected, setSelected] = useState<Set<SelectionKey>>(new Set());

  // Every click simply toggles that item independently — no shift required.
  const handleSelect = useCallback((key: SelectionKey) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  const clearSelection = useCallback(() => setSelected(new Set()), []);

  // Clear selection when view changes
  useEffect(() => { clearSelection(); }, [viewMode, currentId, clearSelection]);

  // ── Data loading ───────────────────────────────────────────────────────────
  const load = useCallback(async (folderId: number | null, mode: ViewMode) => {
    setLoading(true);
    try {
      if (mode === 'recent') {
        const { data } = await filesApi.recent();
        setFiles(data); setFolders([]);
      } else if (mode === 'starred') {
        const { data } = await filesApi.starred();
        setFiles(data); setFolders([]);
      } else if (mode === 'trash') {
        const [fRes, fiRes] = await Promise.all([foldersApi.listTrashed(), filesApi.trash()]);
        setFolders(fRes.data); setFiles(fiRes.data);
      } else {
        const [fRes, fiRes] = await Promise.all([
          foldersApi.list(folderId),
          filesApi.list(folderId),
        ]);
        setFolders(fRes.data); setFiles(fiRes.data);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(currentId, viewMode); }, [currentId, viewMode, load]);

  const refresh = useCallback(() => load(currentId, viewMode), [load, currentId, viewMode]);

  // ── Filtered lists ─────────────────────────────────────────────────────────
  const filteredFolders = useMemo(() =>
    !search.trim() ? folders : folders.filter(f => f.name.toLowerCase().includes(search.toLowerCase())),
  [folders, search]);

  const filteredFiles = useMemo(() =>
    !search.trim() ? files : files.filter(f => f.file_name.toLowerCase().includes(search.toLowerCase())),
  [files, search]);

  // ── Navigation ─────────────────────────────────────────────────────────────
  const openFolder = (f: Folder) => {
    setViewMode('my-drive');
    setCurrentId(f.id);
    setSearch('');
    setBreadcrumbs(prev => [...prev, { id: f.id, name: f.name }]);
  };

  const goToBreadcrumb = (item: BreadcrumbItem) => {
    setViewMode('my-drive');
    setCurrentId(item.id);
    setSearch('');
    const idx = breadcrumbs.findIndex(b => b.id === item.id);
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

  // ── Individual file/folder actions ─────────────────────────────────────────
  const handleTrashFolder   = async (id: number) => { await foldersApi.trash(id);   refresh(); };
  const handleRestoreFolder = async (id: number) => { await foldersApi.restore(id); refresh(); };
  const handleDeleteFolder  = async (id: number) => { await foldersApi.delete(id);  refresh(); };
  const handleMoveFile      = async (fileId: number, folderId: number | null) => {
    await filesApi.move(fileId, folderId); refresh();
  };
  const handleToggleStar  = async (id: number) => { await filesApi.toggleStar(id);  refresh(); };
  const handleTrashFile   = async (id: number) => { await filesApi.moveToTrash(id); refresh(); };
  const handleRestoreFile = async (id: number) => { await filesApi.restore(id);     refresh(); };
  const handleDeleteFile  = async (id: number) => { await filesApi.delete(id);      refresh(); };

  // ── Bulk actions ───────────────────────────────────────────────────────────
  const selectedFileIds   = [...selected].filter(k => k.startsWith('file-')).map(k => parseInt(k.slice(5)));
  const selectedFolderIds = [...selected].filter(k => k.startsWith('folder-')).map(k => parseInt(k.slice(7)));

  const handleBulkTrash = useCallback(async () => {
    await Promise.all([
      ...selectedFileIds.map(id   => filesApi.moveToTrash(id)),
      ...selectedFolderIds.map(id => foldersApi.trash(id)),
    ]);
    clearSelection();
    refresh();
  }, [selectedFileIds, selectedFolderIds, clearSelection, refresh]);

  const handleBulkRestore = useCallback(async () => {
    await Promise.all([
      ...selectedFileIds.map(id   => filesApi.restore(id)),
      ...selectedFolderIds.map(id => foldersApi.restore(id)),
    ]);
    clearSelection();
    refresh();
  }, [selectedFileIds, selectedFolderIds, clearSelection, refresh]);

  const handleBulkDelete = useCallback(async () => {
    await Promise.all([
      ...selectedFileIds.map(id   => filesApi.delete(id)),
      ...selectedFolderIds.map(id => foldersApi.delete(id)),
    ]);
    clearSelection();
    refresh();
  }, [selectedFileIds, selectedFolderIds, clearSelection, refresh]);

  const handleBulkStar = useCallback(async () => {
    await Promise.all(selectedFileIds.map(id => filesApi.toggleStar(id)));
    clearSelection();
    refresh();
  }, [selectedFileIds, clearSelection, refresh]);

  // ── Upload ─────────────────────────────────────────────────────────────────
  const onUploadProgress = (p: UploadProgress) => {
    setUploads(prev => new Map(prev).set(p.fileUploadId, p));
  };

  const onFileComplete = useCallback(() => {
    load(currentId, viewMode);
  }, [load, currentId, viewMode]);

  useEffect(() => {
    const allDone = uploads.size > 0 && [...uploads.values()].every(u => u.status === 'completed');
    if (!allDone) return;
    const t = setTimeout(() => setUploads(new Map()), 3000);
    return () => clearTimeout(t);
  }, [uploads]);

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      <Sidebar
        activeView={viewMode}
        onViewChange={changeView}
        onNewFolder={() => setShowFolder(true)}
        onNewFile={() => setShowUpload(true)}
      />

      <div className="flex flex-col flex-1 overflow-hidden">
        <Header onSearch={setSearch}/>

        <main className="flex-1 overflow-auto p-6">

          {/* Breadcrumb */}
          {viewMode === 'my-drive' && (
            <nav className="flex items-center gap-1 text-sm mb-6 text-gray-500">
              {breadcrumbs.map((b, i) => (
                <span key={b.id ?? 'root'} className="flex items-center gap-1">
                  {i > 0 && <span className="text-gray-300">/</span>}
                  <button
                    onClick={() => goToBreadcrumb(b)}
                    className={`hover:text-blue-600 transition-colors ${i === breadcrumbs.length - 1 ? 'text-gray-900 font-medium' : ''}`}
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
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4"/>
                </svg>
                New
              </button>
              <button
                onClick={() => setShowFolder(true)}
                className="flex items-center gap-2 px-4 py-2 bg-white border border-gray-200 rounded-full text-sm font-medium text-gray-700 hover:bg-gray-50 shadow-sm transition-colors"
              >
                <svg className="w-4 h-4 text-yellow-500" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M10 4H4c-1.11 0-2 .89-2 2v12c0 1.1.89 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/>
                </svg>
                New Folder
              </button>
            </div>
          )}

          {/* Upload progress strip */}
          {uploads.size > 0 && (
            <div className="mb-6 space-y-2">
              {[...uploads.values()].map(up => (
                <div key={up.fileUploadId} className="bg-white rounded-xl border border-gray-200 px-4 py-3 flex items-center gap-3 shadow-sm">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-700 truncate">{up.fileName}</p>
                    <div className="mt-1.5 h-1.5 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-blue-500 transition-all duration-300 rounded-full"
                        style={{ width: `${up.percentage}%` }}
                      />
                    </div>
                  </div>
                  <span className="text-xs text-gray-400 shrink-0 w-10 text-right">{up.percentage}%</span>
                  {up.status === 'completed' && (
                    <svg className="w-4 h-4 text-green-500 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
                    </svg>
                  )}
                </div>
              ))}
            </div>
          )}

          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"/>
            </div>
          ) : (
            <FileGrid
              folders={filteredFolders}
              files={filteredFiles}
              viewMode={viewMode}
              selected={selected}
              onSelect={handleSelect}
              onClearSelection={clearSelection}
              onOpenFolder={openFolder}
              onTrashFolder={handleTrashFolder}
              onRestoreFolder={handleRestoreFolder}
              onDeleteFolder={handleDeleteFolder}
              onMoveFile={handleMoveFile}
              onToggleStar={handleToggleStar}
              onTrashFile={handleTrashFile}
              onRestoreFile={handleRestoreFile}
              onDeleteFile={handleDeleteFile}
              onBulkTrash={handleBulkTrash}
              onBulkRestore={handleBulkRestore}
              onBulkDelete={handleBulkDelete}
              onBulkStar={handleBulkStar}
            />
          )}
        </main>
      </div>

      {showUpload && (
        <UploadModal
          folderId={currentId}
          onClose={() => setShowUpload(false)}
          onProgress={onUploadProgress}
          onComplete={onFileComplete}
        />
      )}

      {showFolder && (
        <CreateFolderModal
          parentId={currentId}
          onClose={() => setShowFolder(false)}
          onCreated={refresh}
        />
      )}
    </div>
  );
}