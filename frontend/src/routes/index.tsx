import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import ProtectedRoute from '../components/ProtectedRoute';
import Layout from '../components/Layout';
import Login from '../pages/Login';
import Register from '../pages/Register';
import Dashboard from '../pages/Dashboard';
import Memories from '../pages/Memories';
import Chat from '../pages/Chat';
import Settings from '../pages/Settings';
import Unified from '../pages/Unified';

export default function AppRoutes() {
  const { user } = useAuth();

  return (
    <Routes>
      <Route path="/login" element={!user ? <Login /> : <Navigate to="/memories" />} />
      <Route path="/register" element={!user ? <Register /> : <Navigate to="/memories" />} />
      <Route element={<ProtectedRoute />}>
        <Route element={<Layout />}>
          <Route path="/unified" element={<Unified />} />
          <Route path="/todos" element={<Dashboard />} />
          <Route path="/memories" element={<Memories />} />
          <Route path="/chat" element={<Chat />} />
          <Route path="/dashboard" element={<Navigate to="/memories" />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to={user ? '/memories' : '/login'} />} />
    </Routes>
  );
}