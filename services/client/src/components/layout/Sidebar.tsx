import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
    MessageCircle,
    FolderOpen,
    CheckSquare,
    Network,
    User,
    Clock,
    StickyNote,
    Zap,
    LucideIcon
} from 'lucide-react';

interface SidebarItem {
    id: string;
    label: string;
    icon: LucideIcon;
    path: string;
    badge?: number;
}

const sidebarItems: SidebarItem[] = [
    {
        id: 'conversation',
        label: 'Conversation',
        icon: MessageCircle,
        path: '/dashboard',
    },
    {
        id: 'projects',
        label: 'Projects',
        icon: FolderOpen,
        path: '/projects',
        badge: 3,
    },
    {
        id: 'tasks',
        label: 'Tasks',
        icon: CheckSquare,
        path: '/tasks',
        badge: 7,
    },
    {
        id: 'connections',
        label: 'Connections',
        icon: Network,
        path: '/connections',
    },
    {
        id: 'notes',
        label: 'Notes',
        icon: StickyNote,
        path: '/notes',
    },
    {
        id: 'recent',
        label: 'Recent Activity',
        icon: Clock,
        path: '/recent',
    },
];

export const Sidebar: React.FC = () => {
    const navigate = useNavigate();
    const location = useLocation();

    const isActive = (path: string) => {
        if (path === '/dashboard') {
            return location.pathname === '/' || location.pathname === '/dashboard';
        }
        return location.pathname.startsWith(path);
    };

    return (
        <aside className="sidebar">
            <div className="sidebar-content">
                <nav className="sidebar-nav">
                    <div className="nav-section">
                        <h3 className="nav-section-title">Main</h3>
                        <ul className="nav-list">
                            {sidebarItems.map((item) => (
                                <li key={item.id}>
                                    <button
                                        className={`nav-item ${isActive(item.path) ? 'active' : ''}`}
                                        onClick={() => navigate(item.path)}
                                    >
                                        <item.icon size={18} />
                                        <span className="nav-label">{item.label}</span>
                                        {item.badge && (
                                            <span className="nav-badge">{item.badge}</span>
                                        )}
                                    </button>
                                </li>
                            ))}
                        </ul>
                    </div>

                    <div className="nav-section">
                        <h3 className="nav-section-title">Quick Actions</h3>
                        <ul className="nav-list">
                            <li>
                                <button className="nav-item">
                                    <Zap size={18} />
                                    <span className="nav-label">New Memory</span>
                                </button>
                            </li>
                            <li>
                                <button className="nav-item">
                                    <FolderOpen size={18} />
                                    <span className="nav-label">New Project</span>
                                </button>
                            </li>
                        </ul>
                    </div>
                </nav>
            </div>
        </aside>
    );
};
