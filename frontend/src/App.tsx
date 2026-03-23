import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import LoginPage    from './components/auth/LoginPage';
import RegisterPage from './components/auth/RegisterPage';
import DrivePage    from './components/drive/DrivePage';

// Только для залогиненных — иначе на /login
function PrivateRoute({ children }: { children: React.ReactNode }) {
  const ok = useAuthStore(s => s.isAuthenticated());
  return ok ? <>{children}</> : <Navigate to="/login" replace />;
}

// Только для незалогиненных — если уже вошёл, на /
function PublicRoute({ children }: { children: React.ReactNode }) {
  const ok = useAuthStore(s => s.isAuthenticated());
  return ok ? <Navigate to="/" replace /> : <>{children}</>;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<PrivateRoute><DrivePage /></PrivateRoute>} />
        <Route path="/login"    element={<PublicRoute><LoginPage /></PublicRoute>} />
        <Route path="/register" element={<PublicRoute><RegisterPage /></PublicRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}