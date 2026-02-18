import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/authStore';
import LoginPage    from './components/auth/LoginPage';
import RegisterPage from './components/auth/RegisterPage';
import DrivePage    from './components/drive/DrivePage';

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const ok = useAuthStore(s => s.isAuthenticated());
  return ok ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login"    element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          path="/drive"
          element={<PrivateRoute><DrivePage /></PrivateRoute>}
        />
        <Route path="*" element={<Navigate to="/drive" replace />} />
      </Routes>
    </BrowserRouter>
  );
}