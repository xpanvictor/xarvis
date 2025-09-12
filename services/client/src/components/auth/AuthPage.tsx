import React, { useState } from 'react';
import { LoginForm, RegisterForm } from './AuthForms';
import './Auth.css';

interface AuthPageProps {
    onSuccess?: () => void;
}

export const AuthPage: React.FC<AuthPageProps> = ({ onSuccess }) => {
    const [activeTab, setActiveTab] = useState<'login' | 'register'>('login');

    return (
        <div className="auth-container">
            <div className="auth-card">
                <div className="auth-tabs">
                    <button
                        className={`auth-tab ${activeTab === 'login' ? 'active' : ''}`}
                        onClick={() => setActiveTab('login')}
                    >
                        Sign In
                    </button>
                    <button
                        className={`auth-tab ${activeTab === 'register' ? 'active' : ''}`}
                        onClick={() => setActiveTab('register')}
                    >
                        Sign Up
                    </button>
                </div>

                {activeTab === 'login' ? (
                    <LoginForm onSuccess={onSuccess} />
                ) : (
                    <RegisterForm onSuccess={onSuccess} />
                )}
            </div>
        </div>
    );
};

export default AuthPage;
