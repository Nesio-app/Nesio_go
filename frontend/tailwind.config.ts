import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        nesio: {
          bg: '#F2F0ED',
          card: '#FFFFFF',
          ink: '#2D2D2D',
          muted: '#8A8A8A',
          accent: '#B87A72',
          accentLight: '#E8D5D1',
          accentSoft: '#F5EBE8',
          border: '#E8E4E0',
          iconBg: '#F5F0EC',
          tabBar: '#FFFFFF',
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
