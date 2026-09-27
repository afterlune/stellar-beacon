module.exports = {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      spacing: {
        97: '26rem',
        98: '28rem',
        '12/10': '120%'
      },
      colors: {
        ob: 'var(--text-accent)',
        'ob-normal': 'var(--text-normal)',
        'ob-trans': 'var(--background-trans)',
        'ob-accent-55': 'var(--bg-accent-55)',
        'ob-secondary': 'var(--text-sub-accent)',
        'ob-bright': 'var(--text-bright)',
        'ob-dim': 'var(--text-dim)',
        'ob-surface': 'var(--surface-1)',
        'ob-surface-2': 'var(--surface-2)',
        'ob-hover': 'var(--surface-hover)',
        'ob-hairline': 'var(--border-hairline)',
        'ob-deep': {
          800: 'var(--background-secondary)',
          900: 'var(--background-primary)'
        }
      },
      fontFamily: {
        sans: ['var(--font-sans)'],
        mono: ['var(--font-mono)']
      },
      boxShadow: {
        ob: 'var(--accent-shadow)',
        'elev-1': 'var(--elev-1)',
        'elev-2': 'var(--elev-2)',
        'elev-3': 'var(--elev-3)'
      }
    }
  },
  variants: {
    extend: {}
  },
  plugins: []
}
