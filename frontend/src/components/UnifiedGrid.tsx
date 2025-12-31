import { motion } from 'framer-motion';
import { Plus } from '@phosphor-icons/react';
import UnifiedCard from './UnifiedCard';
import type { Memory, Todo, RAGSearchResult } from '../types';

interface UnifiedGridProps {
  items: (Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean })[];
  type: 'memory' | 'todo';
  onCreateClick: () => void;
  onItemClick: (item: Memory | Todo | { document: RAGSearchResult['document']; isCitation?: boolean }) => void;
}

export default function UnifiedGrid({ items, type, onCreateClick, onItemClick }: UnifiedGridProps) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      {/* Create Memory Card - Always first */}
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        onClick={onCreateClick}
        className="bg-white dark:bg-gray-900 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-700 p-6 hover:border-primary-500 dark:hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10 transition-all cursor-pointer flex flex-col items-center justify-center min-h-[120px]"
      >
        <Plus size={32} weight="bold" className="text-gray-400 dark:text-gray-500 mb-2" />
        <p className="text-sm font-medium text-gray-600 dark:text-gray-400">
          Create Memory
        </p>
      </motion.div>

      {/* Items Grid */}
      {items.map((item, index) => {
        const itemId = 'id' in item ? item.id : ('document' in item ? item.document.id : `citation-${index}`);
        return (
          <UnifiedCard
            key={itemId}
            item={item}
            type={type}
            onClick={() => onItemClick(item)}
          />
        );
      })}
    </div>
  );
}

