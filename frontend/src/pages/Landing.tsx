import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { ArrowRight } from '@phosphor-icons/react';
import { useEffect } from 'react';

function Hero() {
  return (
    <section className="min-h-screen flex items-center justify-center px-6 py-20 bg-gradient-to-br from-primary-400/10 via-primary-500/5 to-secondary-500/10 dark:from-primary-900/20 dark:via-primary-800/10 dark:to-secondary-900/20">
      <div className="max-w-4xl mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="text-center"
        >
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.05 }}
            className="flex items-center justify-center gap-3 mb-6"
          >
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center">
              <img src="/logo-white.png" alt="Mr.Brain" className="w-6 h-6" />
            </div>
            <span className="text-xl font-heading text-gray-900 dark:text-white">memlane</span>
          </motion.div>
          <motion.img
            src="/peek.png"
            alt="Peek"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
            className="mx-auto mb-0 max-w-xs md:max-w-sm"
          />
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-heading text-gray-900 dark:text-white mb-6 leading-tight">
            I've been forgetting important things for years
          </h1>
          
          <p className="text-xl md:text-2xl text-gray-700 dark:text-gray-300 mb-8 max-w-2xl mx-auto leading-relaxed">
            Movie suggestions from friends. Personal improvement notes. That solution I thought of in the shower. 
          </p>
          
          <p className="text-lg text-gray-600 dark:text-gray-400 mb-8 max-w-xl mx-auto">
            So I built memlane. It's my second brain – everything I capture gets organized, indexed, and searchable. 
            I can ask it questions. It remembers what I forget.
          </p>

          {/* Imposition text */}


          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="flex flex-col sm:flex-row gap-4 justify-center items-center"
          >
            <Link
              to="/register"
              className="btn-primary flex items-center gap-2 text-lg px-8 py-4"
            >
              <span>Start Remembering</span>
              <ArrowRight size={24} weight="regular" />
            </Link>
            <Link
              to="/login"
              className="px-8 py-4 text-lg font-medium text-gray-700 dark:text-gray-300 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
            >
              Sign in
            </Link>
          </motion.div>
        </motion.div>
      </div>
    </section>
  );
}

function SolutionSection() {
  return (
    <section className="py-20 px-6 bg-surface-light dark:bg-surface-dark">
      <div className="max-w-4xl mx-auto">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
        >
          <p className="text-xl md:text-2xl text-gray-900 dark:text-white mb-12 font-medium text-center">
            If I don't write it down, it's gone forever.
          </p>

          <h2 className="text-3xl md:text-4xl font-heading text-gray-900 dark:text-white mb-6">
            How it works
          </h2>
          
          <div className="space-y-8 text-lg text-gray-700 dark:text-gray-300 leading-relaxed">
            <p>
              You capture things. Ideas, notes, links, movie suggestions – whatever. Just type it in. 
              No forms, no categories to pick from. Just capture.
            </p>
            
            <p>
              AI automatically organizes everything. It categorizes your memories – movies, books, ideas, 
              personal goals. It extracts the important bits from URLs. It even detects when you're asking 
              a question and fetches answers for you.
            </p>
            
            <p>
              Everything is searchable. Not just keyword search – semantic search. Ask "what movies did 
              I want to watch?" and it finds them, even if you never used the word "movie" when you saved it.
            </p>
            
            <p>
              You can chat with it. Ask questions about your notes and memories. It uses everything you've 
              saved to give you answers. It's like having a conversation with your past self.
            </p>
            
            <p className="pt-4 border-t border-gray-200 dark:border-gray-800">
              That's it. No complicated setup. No integrations. Just capture, organize, search, and ask. 
              2026 is the year I stop forgetting important shit.
            </p>
          </div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.6, delay: 0.2 }}
            className="mt-12 flex flex-col sm:flex-row gap-4 justify-center items-center"
          >
            <Link
              to="/register"
              className="btn-primary inline-flex items-center gap-2 text-lg px-8 py-4"
            >
              <span>Try it</span>
              <ArrowRight size={24} weight="regular" />
            </Link>
            <a
              href="https://github.com/entrefix/mrbrain"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-lg px-8 py-4 font-medium text-gray-700 dark:text-gray-300 hover:text-primary-600 dark:hover:text-primary-400 transition-colors border border-gray-300 dark:border-gray-700 rounded-lg hover:border-primary-500 dark:hover:border-primary-500"
            >
              <svg 
                height="24" 
                width="24" 
                viewBox="0 0 24 24" 
                fill="currentColor"
                aria-hidden="true"
                className="w-6 h-6"
              >
                <path d="M12 1C5.923 1 1 5.923 1 12c0 4.867 3.149 8.979 7.521 10.436.55.096.756-.233.756-.522 0-.262-.013-1.128-.013-2.049-2.764.509-3.479-.674-3.699-1.292-.124-.317-.66-1.293-1.127-1.554-.385-.207-.936-.715-.014-.729.866-.014 1.485.797 1.691 1.128.99 1.663 2.571 1.196 3.204.907.096-.715.385-1.196.701-1.471-2.448-.275-5.005-1.224-5.005-5.432 0-1.196.426-2.186 1.128-2.956-.111-.275-.496-1.402.11-2.915 0 0 .921-.288 3.024 1.128a10.193 10.193 0 0 1 2.75-.371c.936 0 1.871.123 2.75.371 2.104-1.43 3.025-1.128 3.025-1.128.605 1.513.221 2.64.111 2.915.701.77 1.127 1.747 1.127 2.956 0 4.222-2.571 5.157-5.019 5.432.399.344.743 1.004.743 2.035 0 1.471-.014 2.654-.014 3.025 0 .289.206.632.756.522C19.851 20.979 23 16.854 23 12c0-6.077-4.922-11-11-11Z"></path>
              </svg>
              <span>Contribute</span>
            </a>
          </motion.div>
        </motion.div>
      </div>
    </section>
  );
}

export default function Landing() {
  useEffect(() => {
    // Enable scrolling on the landing page
    document.body.style.position = 'relative';
    document.body.style.overflow = 'auto';
    document.body.style.height = 'auto';
    document.documentElement.style.overflow = 'auto';
    document.documentElement.style.height = 'auto';
    
    // Cleanup: restore original styles when component unmounts
    return () => {
      document.body.style.position = 'fixed';
      document.body.style.overflow = 'hidden';
      document.body.style.height = '100%';
      document.documentElement.style.overflow = 'hidden';
      document.documentElement.style.height = '100%';
    };
  }, []);

  return (
    <div className="min-h-screen">
      {/* Main content */}
      <main>
        <Hero />
        <SolutionSection />
      </main>

      {/* Footer */}
      <footer className="py-12 px-6 bg-surface-light dark:bg-surface-dark border-t border-gray-200/50 dark:border-gray-800/50">
        <div className="max-w-6xl mx-auto text-center">
          <div className="flex items-center justify-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center">
              <img src="/logo-white.png" alt="Mr.Brain" className="w-6 h-6" />
            </div>
            <span className="text-lg font-heading text-gray-900 dark:text-white">memlane</span>
          </div>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            Your second brain for 2026. Never forget another idea.
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-500">
            © 2026 memlane. Built to help you remember what matters.
          </p>
        </div>
      </footer>
    </div>
  );
}

