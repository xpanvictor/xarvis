import React from 'react';
import { ChatInterface } from './components/ChatInterface';
import { Header } from './components/Header';
import './App.css';

const App: React.FC = () => {
  return (
    <div className="app">
      <Header />
      <main className="main-content">
        <ChatInterface />
      </main>
    </div>
  );
};

export default App;
