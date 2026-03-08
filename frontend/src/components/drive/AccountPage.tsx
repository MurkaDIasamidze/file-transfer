import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import { api } from '../../services/api';

interface Props {
  onClose: () => void;
}

export default function AccountPage({ onClose }: Props) {
  const navigate = useNavigate();
  const { user, logout, setAuth, token } = useAuthStore();
  const [tab, setTab] = useState<'profile' | 'security'>('profile');
  const [name, setName] = useState(user?.name ?? '');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [pwdForm, setPwdForm] = useState({ current: '', next: '', confirm: '' });
  const [pwdError, setPwdError] = useState('');
  const [pwdOk, setPwdOk] = useState(false);

  const initials = user?.name?.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase() ?? '?';

  const handleSaveProfile = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      const { data } = await api.patch('/api/me', { name: name.trim() });
      setAuth(data, token!);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    if (pwdForm.next !== pwdForm.confirm) { setPwdError("Passwords don't match"); return; }
    if (pwdForm.next.length < 6) { setPwdError("Password must be at least 6 characters"); return; }
    setPwdError('');
    setSaving(true);
    try {
      await api.post('/api/me/password', {
        current_password: pwdForm.current,
        new_password:     pwdForm.next,
      });
      setPwdOk(true);
      setPwdForm({ current: '', next: '', confirm: '' });
      setTimeout(() => setPwdOk(false), 2000);
    } catch (err) {
      const e = err as { response?: { data?: { error?: string } } };
      setPwdError(e.response?.data?.error ?? 'Failed to update password');
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm p-4">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md overflow-hidden">
        {/* Header */}
        <div className="bg-gradient-to-br from-blue-600 to-indigo-700 px-6 pt-8 pb-16 relative">
          <button
            onClick={onClose}
            className="absolute top-4 right-4 w-8 h-8 flex items-center justify-center rounded-full bg-white/20 hover:bg-white/30 transition-colors text-white"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-2xl bg-white/20 backdrop-blur flex items-center justify-center text-white text-2xl font-bold border-2 border-white/30 shadow-lg">
              {initials}
            </div>
            <div>
              <h2 className="text-white text-xl font-bold">{user?.name}</h2>
              <p className="text-blue-200 text-sm mt-0.5">{user?.email}</p>
              <p className="text-blue-300 text-xs mt-1">
                Member since {user?.created_at ? new Date(user.created_at).toLocaleDateString('en-US', { month: 'long', year: 'numeric' }) : '—'}
              </p>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="px-6 -mt-6 relative z-10">
          <div className="flex gap-1 bg-white rounded-xl shadow-md p-1 border border-gray-100">
            {(['profile', 'security'] as const).map(t => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`flex-1 py-2 rounded-lg text-sm font-medium transition-all capitalize ${
                  tab === t ? 'bg-blue-600 text-white shadow-sm' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {t === 'profile' ? '👤 Profile' : '🔒 Security'}
              </button>
            ))}
          </div>
        </div>

        {/* Body */}
        <div className="p-6 space-y-4 mt-2">
          {tab === 'profile' && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Full name</label>
                <input
                  value={name}
                  onChange={e => setName(e.target.value)}
                  className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm bg-gray-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Email</label>
                <input
                  value={user?.email ?? ''}
                  disabled
                  className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm bg-gray-100 text-gray-400 cursor-not-allowed"
                />
                <p className="text-xs text-gray-400 mt-1">Email cannot be changed</p>
              </div>
              <button
                onClick={handleSaveProfile}
                disabled={saving || name === user?.name}
                className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-xl text-sm font-medium transition-colors flex items-center justify-center gap-2"
              >
                {saving ? 'Saving…' : saved ? '✓ Saved' : 'Save changes'}
              </button>
            </>
          )}

          {tab === 'security' && (
            <>
              {['current', 'next', 'confirm'].map((key) => (
                <div key={key}>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5 capitalize">
                    {key === 'next' ? 'New password' : key === 'confirm' ? 'Confirm new password' : 'Current password'}
                  </label>
                  <input
                    type="password"
                    value={pwdForm[key as keyof typeof pwdForm]}
                    onChange={e => setPwdForm(p => ({ ...p, [key]: e.target.value }))}
                    className="w-full px-4 py-2.5 border border-gray-200 rounded-xl text-sm bg-gray-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                    placeholder="••••••••"
                  />
                </div>
              ))}
              {pwdError && <p className="text-sm text-red-600 bg-red-50 px-3 py-2 rounded-xl">{pwdError}</p>}
              {pwdOk && <p className="text-sm text-green-700 bg-green-50 px-3 py-2 rounded-xl">✓ Password updated</p>}
              <button
                onClick={handleChangePassword}
                disabled={saving || !pwdForm.current || !pwdForm.next}
                className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-xl text-sm font-medium transition-colors"
              >
                {saving ? 'Updating…' : 'Update password'}
              </button>
            </>
          )}

          {/* Danger zone */}
          <div className="border-t border-gray-100 pt-4 mt-2">
            <button
              onClick={handleLogout}
              className="w-full py-2.5 border border-red-200 text-red-600 hover:bg-red-50 rounded-xl text-sm font-medium transition-colors flex items-center justify-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
              </svg>
              Sign out
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}