import React from 'react';
import { Bot } from 'lucide-react';

export const Header: React.FC = () => {
  return (
    <header className="header">
      <div className="header-content">
        <div className="logo">
          <Bot size={24} />
          <span>Xarvis</span>
        </div>
        <div className="status-indicator">
          <div className="status-dot"></div>
          <span>Online</span>
        </div>
      </div>
    </header>
  );
};
