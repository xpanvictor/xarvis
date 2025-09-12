import React, { useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthPage } from '../components/auth/AuthPage';
import { Layout } from '../components/layout/Layout';
import { ProtectedRoute, PublicRoute } from '../components/layout/ProtectedRoute';
import { Dashboard } from '../pages/Dashboard';
import { ProjectsPage, TasksPage, ConnectionsPage } from '../pages/PlaceholderPages';
import { useAuthStore } from '../store';
import { isAuthenticated, getCurrentUser } from '../services/api';
import '../pages/Pages.css';
import './App.css';

const App: React.FC = () => {
  const { setUser } = useAuthStore();

  // Check authentication status on app load
  useEffect(() => {
    const checkAuth = () => {
      if (isAuthenticated()) {
        const user = getCurrentUser();
        if (user) {
          setUser(user);
        }
      }
    };

    checkAuth();
  }, [setUser]);

  return (
    <Router>
      <div className="app">
        <Routes>
          {/* Public routes */}
          <Route
            path="/auth"
            element={
              <PublicRoute>
                <AuthPage />
              </PublicRoute>
            }
          />

          {/* Protected routes with layout */}
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<Dashboard />} />
            <Route path="projects" element={<ProjectsPage />} />
            <Route path="tasks" element={<TasksPage />} />
            <Route path="connections" element={<ConnectionsPage />} />
            <Route path="analytics" element={<div className="page-container"><div className="empty-state"><h3>Analytics Coming Soon</h3></div></div>} />
            <Route path="recent" element={<div className="page-container"><div className="empty-state"><h3>Recent Activity Coming Soon</h3></div></div>} />
            <Route path="profile" element={<div className="page-container"><div className="empty-state"><h3>Profile Coming Soon</h3></div></div>} />
            <Route path="settings" element={<div className="page-container"><div className="empty-state"><h3>Settings Coming Soon</h3></div></div>} />
          </Route>

          {/* Catch-all redirect */}
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </div>
    </Router>
  );
};

export default App;
