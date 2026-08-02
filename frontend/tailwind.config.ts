import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        nesio: {
          bg: 'var(--nesio-bg)',
          card: 'var(--nesio-card)',
          ink: 'var(--nesio-ink)',
          muted: 'var(--nesio-muted)',
          accent: 'var(--nesio-accent)',
          accentLight: 'var(--nesio-accent-light)',
          accentSoft: 'var(--nesio-accent-soft)',
          border: 'var(--nesio-border)',
          iconBg: 'var(--nesio-icon-bg)',
          tabBar: 'var(--nesio-card)',
        }
      },
      fontFamily: {
        sans: ['"Noto Sans SC"', 'system-ui', 'sans-serif'],
      },
      borderRadius: {
        '2xl': '18px',
        '3xl': '24px',
      },
      boxShadow: {
        'card': '0 2px 12px rgba(0,0,0,0.04)',
        'float': '0 4px 20px rgba(0,0,0,0.08)',
      }
    },
  },
  plugins: [],
} satisfies Config
