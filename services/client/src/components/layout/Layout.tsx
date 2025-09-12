import React from 'react';
import { Outlet } from 'react-router-dom';
import { Navbar } from './Navbar';
import { Sidebar } from './Sidebar';
import { useUIStore } from '../../store';
import './Layout.css';

export const Layout: React.FC = () => {
    const { sidebarOpen } = useUIStore();

    return (
        <div className="layout">
            <Navbar />
            <div className="layout-body">
                {sidebarOpen && <Sidebar />}
                <main className={`main-content ${sidebarOpen ? 'with-sidebar' : 'full-width'}`}>
                    <Outlet />
                </main>
            </div>
        </div>
    );
};

export default Layout;
