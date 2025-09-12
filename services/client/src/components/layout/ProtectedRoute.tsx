import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuthStore } from '../../store';
import { isAuthenticated } from '../../services/api';

interface ProtectedRouteProps {
    children: React.ReactNode;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
    const { isAuthenticated: storeAuth } = useAuthStore();

    // Check both store state and token validity
    const authenticated = storeAuth && isAuthenticated();

    if (!authenticated) {
        return <Navigate to="/auth" replace />;
    }

    return <>{children}</>;
};

interface PublicRouteProps {
    children: React.ReactNode;
}

export const PublicRoute: React.FC<PublicRouteProps> = ({ children }) => {
    const { isAuthenticated: storeAuth } = useAuthStore();

    // Check both store state and token validity
    const authenticated = storeAuth && isAuthenticated();

    if (authenticated) {
        return <Navigate to="/dashboard" replace />;
    }

    return <>{children}</>;
};
