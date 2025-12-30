import { Link, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { CheckSquare, BookmarkSimple, ChatCircle, Gear, SignOut, Sun, Moon, X } from '@phosphor-icons/react';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function Sidebar({ isOpen, onClose }: SidebarProps) {
  const location = useLocation();
  const { signOut } = useAuth();
  const { theme, toggleTheme } = useTheme();

  const isActive = (path: string) => location.pathname === path;

  const handleNavClick = () => {
    // Only close on mobile
    if (window.innerWidth < 1024) {
      onClose();
    }
  };

  const navItems = [
    { path: '/todos', icon: CheckSquare, label: 'Todos' },
    { path: '/memories', icon: BookmarkSimple, label: 'Memories' },
    { path: '/chat', icon: ChatCircle, label: 'Ask AI' },
    { path: '/settings', icon: Gear, label: 'Settings' },
  ];

  const SidebarContent = () => (
    <div className="h-full flex flex-col">
      <div className="p-6 flex justify-between items-center">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center">
            <span className="text-white text-sm font-bold">M</span>
          </div>
          <h1 className="text-xl font-heading text-gray-800 dark:text-white">Mr. Brain</h1>
        </div>
        <button
          onClick={onClose}
          className="lg:hidden p-2 rounded-xl hover:bg-gray-100 dark:hover:bg-surface-dark-muted transition-colors"
        >
          <X size={20} className="text-gray-500 dark:text-gray-400" />
        </button>
      </div>

      <nav className="flex-1 px-3">
        {navItems.map(({ path, icon: Icon, label }) => (
          <Link
            key={path}
            to={path}
            onClick={handleNavClick}
            className={`flex items-center px-4 py-3 mb-1.5 rounded-xl transition-all duration-200 ${
              isActive(path)
                ? 'bg-primary-100 dark:bg-primary-900/40 text-primary-600 dark:text-primary-400 font-medium shadow-subtle'
                : 'text-gray-600 dark:text-gray-400 hover:bg-surface-light-muted dark:hover:bg-surface-dark-muted hover:text-gray-900 dark:hover:text-gray-200'
            }`}
          >
            <Icon size={20} weight="regular" className="mr-3" />
            {label}
          </Link>
        ))}
      </nav>

      <div className="p-3 border-t border-gray-100 dark:border-gray-800/50">
        <button
          onClick={toggleTheme}
          className="flex items-center px-4 py-3 w-full rounded-xl text-gray-600 dark:text-gray-400 hover:bg-surface-light-muted dark:hover:bg-surface-dark-muted hover:text-gray-900 dark:hover:text-gray-200 transition-all duration-200"
        >
          {theme === 'dark' ? (
            <Sun size={20} weight="regular" className="mr-3" />
          ) : (
            <Moon size={20} weight="regular" className="mr-3" />
          )}
          {theme === 'dark' ? 'Light Mode' : 'Dark Mode'}
        </button>

        <button
          onClick={signOut}
          className="flex items-center px-4 py-3 w-full rounded-xl text-red-500 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-all duration-200"
        >
          <SignOut size={20} weight="regular" className="mr-3" />
          Sign Out
        </button>
      </div>
    </div>
  );

  return (
    <>
      {/* Desktop Sidebar - Glass effect, in flow */}
      <aside className="hidden lg:flex w-64 flex-shrink-0 bg-white/70 dark:bg-surface-dark/70 backdrop-blur-glass rounded-2xl shadow-subtle border border-white/50 dark:border-gray-800/30 overflow-hidden">
        <SidebarContent />
      </aside>

      {/* Mobile Sidebar - Overlay with glass */}
      <AnimatePresence>
        {isOpen && (
          <>
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 bg-black/40 backdrop-blur-sm z-40 lg:hidden"
              onClick={onClose}
            />

            {/* Sidebar panel */}
            <motion.aside
              initial={{ x: -280 }}
              animate={{ x: 0 }}
              exit={{ x: -280 }}
              transition={{ duration: 0.25, ease: 'easeOut' }}
              className="fixed top-3 bottom-3 left-3 w-64 bg-white/90 dark:bg-surface-dark/90 backdrop-blur-glass rounded-2xl shadow-float border border-white/50 dark:border-gray-800/30 z-50 lg:hidden overflow-hidden"
            >
              <SidebarContent />
            </motion.aside>
          </>
        )}
      </AnimatePresence>
    </>
  );
}
