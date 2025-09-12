import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
    Brain,
    User,
    Settings,
    LogOut,
    Menu,
    X,
    Bell,
    Search
} from 'lucide-react';
import { useAuthStore, useUIStore } from '../../store';
import { authAPI } from '../../services/api';

export const Navbar: React.FC = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const { user, logout } = useAuthStore();
    const { sidebarOpen, setSidebarOpen } = useUIStore();

    const handleLogout = () => {
        authAPI.logout();
        logout();
        navigate('/auth');
    };

    const toggleSidebar = () => {
        setSidebarOpen(!sidebarOpen);
    };

    return (
        <nav className="navbar">
            <div className="navbar-left">
                <button
                    className="sidebar-toggle"
                    onClick={toggleSidebar}
                    aria-label="Toggle sidebar"
                >
                    {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
                </button>

                <div className="logo" onClick={() => navigate('/dashboard')}>
                    <Brain size={24} />
                    <span>Xarvis</span>
                </div>
            </div>

            <div className="navbar-center">
                <div className="search-container">
                    <Search className="search-icon" size={16} />
                    <input
                        type="text"
                        placeholder="Search memories, conversations..."
                        className="search-input"
                    />
                </div>
            </div>

            <div className="navbar-right">
                <button className="nav-icon-button" aria-label="Notifications">
                    <Bell size={18} />
                </button>

                <div className="user-menu">
                    <button className="user-button">
                        <User size={18} />
                        <span className="user-name">{user?.displayName || 'User'}</span>
                    </button>

                    <div className="user-dropdown">
                        <button
                            className="dropdown-item"
                            onClick={() => navigate('/profile')}
                        >
                            <User size={16} />
                            <span>Profile</span>
                        </button>
                        <button
                            className="dropdown-item"
                            onClick={() => navigate('/settings')}
                        >
                            <Settings size={16} />
                            <span>Settings</span>
                        </button>
                        <hr className="dropdown-divider" />
                        <button
                            className="dropdown-item logout"
                            onClick={handleLogout}
                        >
                            <LogOut size={16} />
                            <span>Sign Out</span>
                        </button>
                    </div>
                </div>
            </div>
        </nav>
    );
};
