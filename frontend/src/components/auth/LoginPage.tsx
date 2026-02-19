import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { authApi } from '../../services/api';
import { useAuthStore } from '../../store/authStore';

interface FieldErrors {
  email?:    string;
  password?: string;
}

export default function LoginPage() {
  const navigate    = useNavigate();
  const { setAuth } = useAuthStore();
  const [form,        setForm]        = useState({ email: '', password: '' });
  const [error,       setError]       = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [loading,     setLoading]     = useState(false);
  const [touched,     setTouched]     = useState<Partial<Record<keyof typeof form, boolean>>>({});
  const [showPwd,     setShowPwd]     = useState(false);

  const validate = (f: typeof form): FieldErrors => {
    const errs: FieldErrors = {};
    if (!f.email)                            errs.email    = 'Email is required';
    else if (!/\S+@\S+\.\S+/.test(f.email)) errs.email    = 'Enter a valid email';
    if (!f.password)                         errs.password = 'Password is required';
    return errs;
  };

  const handleChange = (key: keyof typeof form, value: string) => {
    const updated = { ...form, [key]: value };
    setForm(updated);
    if (error) setError('');
    if (touched[key]) setFieldErrors(validate(updated));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched({ email: true, password: true });
    const errs = validate(form);
    setFieldErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setError('');
    setLoading(true);
    try {
      const { data } = await authApi.login(form.email, form.password);
      setAuth(data.user, data.token);
      navigate('/drive');
    } catch (err) {
      const e = err as { response?: { data?: { error?: string } } };
      const msg = e.response?.data?.error ?? '';
      if (msg.toLowerCase().includes('email') || msg.toLowerCase().includes('account')) {
        setFieldErrors({ email: msg });
      } else if (msg.toLowerCase().includes('password') || msg.toLowerCase().includes('incorrect')) {
        setFieldErrors({ password: msg });
      } else {
        setError(msg || 'Login failed. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  const inputBase = 'w-full px-4 py-2.5 rounded-xl border text-sm outline-none transition-all focus:ring-2 focus:border-transparent bg-gray-50 focus:bg-white';
  const inputClass = (key: keyof FieldErrors) =>
    `${inputBase} ${touched[key] && fieldErrors[key] ? 'border-red-400 focus:ring-red-400' : 'border-gray-200 focus:ring-blue-500'}`;

  return (
    <div className="min-h-screen flex">
      {/* Left panel - decorative */}
      <div className="hidden lg:flex lg:w-1/2 bg-gradient-to-br from-blue-600 via-blue-700 to-indigo-800 flex-col justify-between p-12 relative overflow-hidden">
        <div className="absolute inset-0 opacity-10">
          {[...Array(6)].map((_, i) => (
            <div
              key={i}
              className="absolute rounded-full border border-white"
              style={{
                width: `${(i + 2) * 120}px`,
                height: `${(i + 2) * 120}px`,
                top: '50%',
                left: '50%',
                transform: 'translate(-50%, -50%)',
              }}
            />
          ))}
        </div>
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-white/20 backdrop-blur rounded-xl flex items-center justify-center">
              <svg className="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M14.5 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V7.5L14.5 2z"/>
                <polyline points="14 2 14 8 20 8"/>
              </svg>
            </div>
            <span className="text-xl font-bold text-white">DriveClone</span>
          </div>
        </div>
        <div className="relative z-10 space-y-4">
          <h2 className="text-4xl font-bold text-white leading-tight">
            Your files,<br/>anywhere.
          </h2>
          <p className="text-blue-200 text-lg leading-relaxed">
            Secure cloud storage with lightning-fast uploads. Access your files from any device, anytime.
          </p>
          <div className="flex items-center gap-6 pt-4">
            {[['256-bit', 'Encryption'], ['99.9%', 'Uptime'], ['∞', 'Access']].map(([val, label]) => (
              <div key={label}>
                <p className="text-white font-bold text-xl">{val}</p>
                <p className="text-blue-200 text-xs">{label}</p>
              </div>
            ))}
          </div>
        </div>
        <div className="relative z-10 text-blue-200 text-sm">
          © {new Date().getFullYear()} DriveClone. All rights reserved.
        </div>
      </div>

      {/* Right panel - form */}
      <div className="flex-1 flex items-center justify-center p-8 bg-gray-50">
        <div className="w-full max-w-sm">
          {/* Mobile logo */}
          <div className="flex items-center gap-2 mb-8 lg:hidden">
            <svg className="w-8 h-8 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14.5 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
            <span className="text-xl font-bold text-gray-900">DriveClone</span>
          </div>

          <div className="mb-8">
            <h1 className="text-2xl font-bold text-gray-900">Welcome back</h1>
            <p className="text-gray-500 mt-1.5 text-sm">Sign in to continue to your drive</p>
          </div>

          <form onSubmit={handleSubmit} noValidate className="space-y-4">
            {error && (
              <div className="flex items-start gap-2.5 bg-red-50 text-red-700 text-sm px-4 py-3 rounded-xl border border-red-100">
                <svg className="w-4 h-4 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd"/>
                </svg>
                {error}
              </div>
            )}

            <div className="space-y-1">
              <label className="block text-sm font-medium text-gray-700">Email</label>
              <input
                type="email"
                value={form.email}
                onChange={e => handleChange('email', e.target.value)}
                onBlur={() => { setTouched(t => ({ ...t, email: true })); setFieldErrors(validate(form)); }}
                className={inputClass('email')}
                placeholder="you@example.com"
                autoComplete="email"
              />
              {touched.email && fieldErrors.email && (
                <p className="text-xs text-red-600 flex items-center gap-1">
                  <svg className="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
                  </svg>
                  {fieldErrors.email}
                </p>
              )}
            </div>

            <div className="space-y-1">
              <label className="block text-sm font-medium text-gray-700">Password</label>
              <div className="relative">
                <input
                  type={showPwd ? 'text' : 'password'}
                  value={form.password}
                  onChange={e => handleChange('password', e.target.value)}
                  onBlur={() => { setTouched(t => ({ ...t, password: true })); setFieldErrors(validate(form)); }}
                  className={`${inputClass('password')} pr-10`}
                  placeholder="••••••••"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPwd(s => !s)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  tabIndex={-1}
                >
                  {showPwd ? (
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"/>
                    </svg>
                  ) : (
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                    </svg>
                  )}
                </button>
              </div>
              {touched.password && fieldErrors.password && (
                <p className="text-xs text-red-600 flex items-center gap-1">
                  <svg className="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd"/>
                  </svg>
                  {fieldErrors.password}
                </p>
              )}
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-xl font-medium text-sm transition-all flex items-center justify-center gap-2 shadow-sm shadow-blue-200"
            >
              {loading && (
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
                </svg>
              )}
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-gray-500">
            Don't have an account?{' '}
            <Link to="/register" className="text-blue-600 hover:text-blue-700 font-medium">Create one</Link>
          </p>
        </div>
      </div>
    </div>
  );
}