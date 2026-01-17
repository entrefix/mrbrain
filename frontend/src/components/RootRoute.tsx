import { useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import Landing from '../pages/Landing';
import Unified from '../pages/Unified';
import Sidebar from './Sidebar';

export default function RootRoute() {
  const { user, loading } = useAuth();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const location = useLocation();
  const isUnifiedPage = location.pathname === '/';

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (user) {
    return (
      <div className={`h-full bg-background-light dark:bg-background-dark ${isUnifiedPage ? '' : 'p-3 lg:p-4'}`}>
        <div className={`flex h-full ${isUnifiedPage ? '' : 'gap-3 lg:gap-4'}`}>
          {/* Sidebar - Hidden on unified page */}
          {!isUnifiedPage && (
            <Sidebar isOpen={isSidebarOpen} onClose={() => setIsSidebarOpen(false)} />
          )}

          {/* Main content */}
          <main className={`${isUnifiedPage ? 'w-full h-full' : 'flex-1'} ${isUnifiedPage ? '' : 'bg-surface-light dark:bg-surface-dark rounded-2xl shadow-subtle border border-gray-200/50 dark:border-gray-800/50'} overflow-hidden relative`}>
            <div className="h-full overflow-hidden">
              {!isUnifiedPage && <div className="lg:hidden h-14" />}
              <Unified />
            </div>
          </main>
        </div>
      </div>
    );
  }

  return <Landing />;
}

