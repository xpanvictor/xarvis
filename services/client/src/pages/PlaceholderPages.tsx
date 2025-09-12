import React from 'react';
import { FolderOpen, Plus, Filter, Search } from 'lucide-react';

export const ProjectsPage: React.FC = () => {
    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Projects</h1>
                    <p>Organize and manage your AI-assisted projects</p>
                </div>
                <div className="header-actions">
                    <button className="action-button">
                        <Plus size={16} />
                        New Project
                    </button>
                </div>
            </div>

            <div className="page-content">
                <div className="empty-state">
                    <FolderOpen size={48} opacity={0.5} />
                    <h3>No Projects Yet</h3>
                    <p>Create your first project to get started with organizing your work.</p>
                    <button className="cta-button">
                        <Plus size={16} />
                        Create Project
                    </button>
                </div>
            </div>
        </div>
    );
};

export const TasksPage: React.FC = () => {
    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Tasks</h1>
                    <p>Track and manage your tasks with AI assistance</p>
                </div>
                <div className="header-actions">
                    <button className="action-button secondary">
                        <Filter size={16} />
                        Filter
                    </button>
                    <button className="action-button">
                        <Plus size={16} />
                        New Task
                    </button>
                </div>
            </div>

            <div className="page-content">
                <div className="empty-state">
                    <div className="empty-icon">✅</div>
                    <h3>No Tasks Yet</h3>
                    <p>Add your first task and let Xarvis help you manage and prioritize your work.</p>
                    <button className="cta-button">
                        <Plus size={16} />
                        Add Task
                    </button>
                </div>
            </div>
        </div>
    );
};

export const ConnectionsPage: React.FC = () => {
    return (
        <div className="page-container">
            <div className="page-header">
                <div className="header-left">
                    <h1>Connections</h1>
                    <p>Manage your network and collaboration connections</p>
                </div>
                <div className="header-actions">
                    <button className="action-button secondary">
                        <Search size={16} />
                        Search
                    </button>
                    <button className="action-button">
                        <Plus size={16} />
                        Add Connection
                    </button>
                </div>
            </div>

            <div className="page-content">
                <div className="empty-state">
                    <div className="empty-icon">🤝</div>
                    <h3>No Connections Yet</h3>
                    <p>Start building your network and connect with others through Xarvis.</p>
                    <button className="cta-button">
                        <Plus size={16} />
                        Add Connection
                    </button>
                </div>
            </div>
        </div>
    );
};
