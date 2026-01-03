import { motion } from 'framer-motion';

interface PillSwitchProps {
  activeTab: 'mems' | 'todos';
  onTabChange: (tab: 'mems' | 'todos') => void;
  isSettingsActive?: boolean;
}

export default function PillSwitch({ activeTab, onTabChange, isSettingsActive = false }: PillSwitchProps) {
  return (
    <div className="flex items-center bg-surface-light-muted dark:bg-surface-dark-muted rounded-full p-1">
      <button
        onClick={() => onTabChange('mems')}
        className={`relative px-4 py-2 rounded-full text-sm font-medium transition-all duration-200 ${
          activeTab === 'mems' && !isSettingsActive
            ? 'text-white'
            : activeTab === 'mems' && isSettingsActive
            ? 'text-primary-600 dark:text-primary-400 font-medium'
            : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
        }`}
      >
        {activeTab === 'mems' && !isSettingsActive && (
          <motion.div
            layoutId="activeTab"
            className="absolute inset-0 bg-primary-600 rounded-full"
            transition={{ type: 'spring', bounce: 0.2, duration: 0.6 }}
          />
        )}
        <span className="relative z-10">Mems</span>
      </button>
      <button
        onClick={() => onTabChange('todos')}
        className={`relative px-4 py-2 rounded-full text-sm font-medium transition-all duration-200 ${
          activeTab === 'todos' && !isSettingsActive
            ? 'text-white'
            : activeTab === 'todos' && isSettingsActive
            ? 'text-primary-600 dark:text-primary-400 font-medium'
            : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
        }`}
      >
        {activeTab === 'todos' && !isSettingsActive && (
          <motion.div
            layoutId="activeTab"
            className="absolute inset-0 bg-primary-600 rounded-full"
            transition={{ type: 'spring', bounce: 0.2, duration: 0.6 }}
          />
        )}
        <span className="relative z-10 flex items-center gap-1.5">
          Todos
          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300">
            Beta
          </span>
        </span>
      </button>
    </div>
  );
}

