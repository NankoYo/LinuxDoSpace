import { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { Footer } from './components/Footer';
import { Home } from './pages/Home';
import { Domains } from './pages/Domains';
import { Settings } from './pages/Settings';
import { Login } from './pages/Login';

export default function App() {
  const [activeTab, setActiveTab] = useState('home');
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDark]);

  const toggleTheme = () => setIsDark(!isDark);

  const renderContent = () => {
    switch (activeTab) {
      case 'home':
        return <Home />;
      case 'domains':
        return <Domains />;
      case 'settings':
        return <Settings />;
      case 'login':
        return <Login />;
      default:
        return <Home />;
    }
  };

  return (
    <div className="min-h-screen font-sans text-gray-900 dark:text-white transition-colors duration-500 overflow-x-hidden relative">
      {/* Background Image */}
      <div 
        className="fixed inset-0 z-[-2] bg-cover bg-center bg-no-repeat transition-all duration-1000 dark:brightness-[0.3]"
        style={{ backgroundImage: 'url(https://www.loliapi.com/acg/)' }}
      />
      
      {/* Overlay for readability */}
      <div className="fixed inset-0 z-[-1] bg-white/40 dark:bg-black/40 backdrop-blur-[2px] transition-colors duration-500" />

      <Navbar 
        activeTab={activeTab} 
        setActiveTab={setActiveTab} 
        isDark={isDark} 
        toggleTheme={toggleTheme} 
      />
      
      <main className="relative z-10 min-h-screen">
        {renderContent()}
      </main>

      <Footer />
    </div>
  );
}
