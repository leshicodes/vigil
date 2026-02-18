import { BrowserRouter, Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { useEffect, useState, useCallback } from 'react';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Timeline from './pages/Timeline';
import Config from './pages/Config';
import Login from './pages/Login';
import ToastContainer from './components/Toast';
import { api, getToken, clearToken } from './lib/api';

type AuthState = 'loading' | 'authenticated' | 'unauthenticated' | 'no-auth';

function AppRoutes() {
  const [authState, setAuthState] = useState<AuthState>('loading');
  const navigate = useNavigate();

  const checkAuth = useCallback(async () => {
    try {
      const res = await api.checkAuth();
      if (!res.auth_enabled) {
        setAuthState('no-auth');
      } else if (res.authenticated) {
        setAuthState('authenticated');
      } else {
        setAuthState('unauthenticated');
      }
    } catch {
      // If checkAuth itself fails, check if we have a token at all.
      if (getToken()) {
        setAuthState('unauthenticated');
      } else {
        // Server might be down or auth might not be enabled.
        // Try loading normally.
        setAuthState('no-auth');
      }
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  const handleLoginSuccess = () => {
    setAuthState('authenticated');
    navigate('/');
  };

  const handleLogout = () => {
    clearToken();
    setAuthState('unauthenticated');
    navigate('/login');
  };

  if (authState === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface-base">
        <div className="w-8 h-8 rounded-full border-2 border-accent-cyan border-t-transparent animate-spin" />
      </div>
    );
  }

  const needsAuth = authState === 'unauthenticated';
  const authEnabled = authState === 'authenticated'; // Only show logout when auth is active

  return (
    <Routes>
      <Route path="/login" element={
        needsAuth
          ? <Login onSuccess={handleLoginSuccess} />
          : <Navigate to="/" replace />
      } />
      <Route element={needsAuth ? <Navigate to="/login" replace /> : <Layout onLogout={authEnabled ? handleLogout : undefined} />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/timeline" element={<Timeline />} />
        <Route path="/config" element={<Config />} />
      </Route>
    </Routes>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
      <ToastContainer />
    </BrowserRouter>
  );
}

export default App;

