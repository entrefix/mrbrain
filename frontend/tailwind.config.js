/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#fff7f5',
          100: '#ffede8',
          200: '#ffd9cf',
          300: '#ffb8a6',
          400: '#ff8c6b', // Main orange/coral - softer
          500: '#ff6b47',
          600: '#f04d2a',
          700: '#c93d1f',
          800: '#a5341c',
          900: '#88301d',
        },
        secondary: {
          50: '#f5f3ff',
          100: '#ede9fe',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
          800: '#5b21b6',
          900: '#4c1d95',
        },
        surface: {
          // Light mode surfaces
          light: '#ffffff',
          'light-elevated': '#ffffff',
          'light-muted': '#faf8f7',
          // Dark mode surfaces (Dark Gray #111)
          dark: '#111111',
          'dark-elevated': '#1a1a1a',
          'dark-muted': '#222222',
        },
        background: {
          // Colored accent backgrounds
          light: '#fff9f7', // Very light orange tint
          dark: '#0d0d0d', // Near black but not pure
        },
      },
      borderRadius: {
        'xl': '0.875rem',
        '2xl': '1rem',
        '3xl': '1.25rem',
      },
      boxShadow: {
        'subtle': '0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.04)',
        'soft': '0 2px 8px -2px rgb(0 0 0 / 0.08), 0 2px 4px -2px rgb(0 0 0 / 0.04)',
        'glass': '0 4px 24px -4px rgb(0 0 0 / 0.08)',
        'float': '0 8px 32px -8px rgb(0 0 0 / 0.12)',
      },
      backdropBlur: {
        'glass': '16px',
      },
      animation: {
        'slide-in': 'slide-in 0.25s ease-out',
        'slide-out': 'slide-out 0.25s ease-in',
        'fade-in': 'fade-in 0.2s ease-out',
        'scale-in': 'scale-in 0.2s ease-out',
        'shimmer': 'shimmer 2s ease-in-out infinite',
        'pulse-soft': 'pulse-soft 2s ease-in-out infinite',
      },
      keyframes: {
        'slide-in': {
          '0%': { transform: 'translateX(-100%)', opacity: '0' },
          '100%': { transform: 'translateX(0)', opacity: '1' },
        },
        'slide-out': {
          '0%': { transform: 'translateX(0)', opacity: '1' },
          '100%': { transform: 'translateX(-100%)', opacity: '0' },
        },
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'scale-in': {
          '0%': { transform: 'scale(0.95)', opacity: '0' },
          '100%': { transform: 'scale(1)', opacity: '1' },
        },
        'shimmer': {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: '0.6' },
          '50%': { opacity: '1' },
        },
      },
      fontWeight: {
        'heading': '600', // Semi-bold for headings
      },
    },
  },
  plugins: [],
}
