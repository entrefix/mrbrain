import { useState } from 'react';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import { List } from '@phosphor-icons/react';

export default function Layout() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  return (
    <div className="min-h-screen bg-background-light dark:bg-background-dark p-3 lg:p-4">
      {/* Mobile menu button */}
      <button
        onClick={() => setIsSidebarOpen(true)}
        className="lg:hidden fixed top-5 left-5 z-40 p-2.5 rounded-xl bg-white/80 dark:bg-surface-dark-elevated/80 backdrop-blur-glass shadow-soft border border-gray-200/50 dark:border-gray-800/50"
      >
        <List size={20} weight="regular" className="text-gray-600 dark:text-gray-300" />
      </button>

      <div className="flex h-[calc(100vh-24px)] lg:h-[calc(100vh-32px)] gap-3 lg:gap-4">
        {/* Sidebar */}
        <Sidebar isOpen={isSidebarOpen} onClose={() => setIsSidebarOpen(false)} />

        {/* Main content */}
        <main className="flex-1 bg-surface-light dark:bg-surface-dark rounded-2xl shadow-subtle border border-gray-200/50 dark:border-gray-800/50 overflow-hidden relative">
          <div className="h-full overflow-y-auto">
            <div className="lg:hidden h-14" /> {/* Spacer for mobile menu button */}
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
