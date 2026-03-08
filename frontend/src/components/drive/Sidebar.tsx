interface Props {
  activeView: 'my-drive' | 'recent' | 'starred' | 'trash';
  onViewChange: (view: 'my-drive' | 'recent' | 'starred' | 'trash') => void;
  onNewFolder: () => void;
  onNewFile: () => void;
}

export default function Sidebar({ activeView, onViewChange, onNewFile }: Props) {
  const items = [
    { icon: '🏠', label: 'My Drive', value: 'my-drive' as const },
    { icon: '🕐', label: 'Recent', value: 'recent' as const },
    { icon: '⭐', label: 'Starred', value: 'starred' as const },
    { icon: '🗑️', label: 'Trash', value: 'trash' as const },
  ];

  return (
    <aside className="w-56 bg-white border-r border-gray-200 flex flex-col shrink-0">
      {/* Logo */}
      <div className="flex items-center gap-2 px-4 h-14 border-b border-gray-200">
        <svg className="w-7 h-7 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="M14.5 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V7.5L14.5 2z" />
          <polyline points="14 2 14 8 20 8" />
        </svg>
        <span className="font-bold text-gray-900">DriveClone</span>
      </div>

      {/* New button */}
      <div className="p-3">
        <button
          onClick={onNewFile}
          className="flex items-center gap-3 w-full px-4 py-2.5 bg-white hover:bg-blue-50 border border-gray-200 hover:border-blue-200 rounded-2xl text-sm font-medium text-gray-700 shadow-sm transition-all"
        >
          <svg className="w-5 h-5 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          New
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-2 py-1 space-y-0.5">
        {items.map((item) => (
          <button
            key={item.value}
            onClick={() => onViewChange(item.value)}
            className={`flex items-center gap-3 w-full px-3 py-2 rounded-full text-sm transition-colors ${
              activeView === item.value ? 'bg-blue-100 text-blue-700 font-medium' : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            <span>{item.icon}</span>
            {item.label}
          </button>
        ))}
      </nav>

      {/* Storage */}
      <div className="p-4 border-t border-gray-200">
        <div className="w-full h-1.5 bg-gray-200 rounded-full mb-2">
          <div className="h-full bg-blue-500 rounded-full" style={{ width: '35%' }} />
        </div>
        <p className="text-xs text-gray-500">3.5 GB of 15 GB used</p>
      </div>
    </aside>
  );
}