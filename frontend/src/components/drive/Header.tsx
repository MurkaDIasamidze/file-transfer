import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';

interface Props {
  onSearch?: (query: string) => void;
}

export default function Header({ onSearch }: Props) {
  const { user, logout } = useAuthStore();
  const navigate = useNavigate();
  const [query, setQuery] = useState('');

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setQuery(val);
    onSearch?.(val);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') {
      setQuery('');
      onSearch?.('');
    }
  };

  return (
    <header className="h-14 bg-white border-b border-gray-200 flex items-center px-4 gap-4 shrink-0">
      <div className="flex-1 max-w-2xl mx-auto">
        <div className="relative">
          <svg className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <circle cx="11" cy="11" r="8"/><path strokeLinecap="round" d="M21 21l-4.35-4.35"/>
          </svg>
          <input
            type="text"
            value={query}
            onChange={handleSearch}
            onKeyDown={handleKeyDown}
            placeholder="Search in Drive"
            className="w-full pl-9 pr-4 py-2 bg-gray-100 hover:bg-gray-200 focus:bg-white focus:ring-2 focus:ring-blue-500 rounded-full text-sm outline-none transition-colors"
          />
          {query && (
            <button
              onClick={() => { setQuery(''); onSearch?.(''); }}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      </div>

      <div className="flex items-center gap-3 ml-auto">
        <span className="text-sm text-gray-600 hidden sm:block">{user?.email}</span>
        <div className="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-white text-sm font-medium">
          {user?.name?.[0]?.toUpperCase() ?? '?'}
        </div>
        <button
          onClick={handleLogout}
          className="text-sm text-gray-500 hover:text-gray-700 px-2 py-1 rounded hover:bg-gray-100 transition-colors"
        >
          Sign out
        </button>
      </div>
    </header>
  );
}