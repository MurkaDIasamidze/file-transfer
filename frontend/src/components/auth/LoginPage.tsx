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

  const validate = (f: typeof form): FieldErrors => {
    const errs: FieldErrors = {};
    if (!f.email)                            errs.email    = 'Email is required';
    else if (!/\S+@\S+\.\S+/.test(f.email)) errs.email    = 'Enter a valid email address';
    if (!f.password)                         errs.password = 'Password is required';
    return errs;
  };

  const handleBlur = (key: keyof typeof form) => {
    setTouched((t) => ({ ...t, [key]: true }));
    setFieldErrors(validate(form));
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
      const e = err as { response?: { data?: { error?: string }; status?: number } };
      const msg = e.response?.data?.error ?? '';

      // Map specific backend messages to field-level errors where possible
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

  const inputClass = (key: keyof FieldErrors) =>
    `w-full px-4 py-2.5 rounded-lg border text-sm outline-none transition-colors focus:ring-2 focus:border-transparent ${
      touched[key] && fieldErrors[key]
        ? 'border-red-400 bg-red-50 focus:ring-red-400'
        : 'border-gray-300 focus:ring-blue-500'
    }`;

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2 mb-4">
            <svg className="w-10 h-10 text-blue-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M14.5 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V7.5L14.5 2z" />
              <polyline points="14 2 14 8 20 8" />
            </svg>
            <span className="text-2xl font-bold text-gray-900">DriveClone</span>
          </div>
          <h1 className="text-xl font-semibold text-gray-700">Sign in to your account</h1>
        </div>

        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
          <form onSubmit={handleSubmit} noValidate className="space-y-5">

            {error && (
              <div className="flex items-start gap-3 bg-red-50 text-red-700 text-sm px-4 py-3 rounded-lg border border-red-200">
                <svg className="w-4 h-4 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">Email</label>
              <input
                type="email"
                value={form.email}
                onChange={(e) => handleChange('email', e.target.value)}
                onBlur={() => handleBlur('email')}
                className={inputClass('email')}
                placeholder="you@example.com"
                autoComplete="email"
              />
              {touched.email && fieldErrors.email && (
                <p className="mt-1.5 text-xs text-red-600 flex items-center gap-1">
                  <svg className="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                  {fieldErrors.email}
                </p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1.5">Password</label>
              <input
                type="password"
                value={form.password}
                onChange={(e) => handleChange('password', e.target.value)}
                onBlur={() => handleBlur('password')}
                className={inputClass('password')}
                placeholder="••••••••"
                autoComplete="current-password"
              />
              {touched.password && fieldErrors.password && (
                <p className="mt-1.5 text-xs text-red-600 flex items-center gap-1">
                  <svg className="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                  {fieldErrors.password}
                </p>
              )}
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg font-medium text-sm transition-colors flex items-center justify-center gap-2"
            >
              {loading && (
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
                </svg>
              )}
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>

          <p className="mt-6 text-center text-sm text-gray-500">
            Don't have an account?{' '}
            <Link to="/register" className="text-blue-600 hover:underline font-medium">Sign up</Link>
          </p>
        </div>
      </div>
    </div>
  );
}