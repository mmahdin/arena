import type { Config } from 'tailwindcss';

/**
 * Tailwind is wired to the design tokens in `src/styles/tokens.css`.
 * Every colour below is a *semantic* token expressed as an RGB triplet so the
 * `/opacity` modifier keeps working. Do not add one-off hex values here —
 * add a token first, then map it.
 */
const semantic = (name: string) => `rgb(var(--s-${name}) / <alpha-value>)`;

export default {
  darkMode: ['selector', '[data-theme="dark"]'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: semantic('bg'),
        surface: semantic('surface'),
        raised: semantic('raised'),
        sunken: semantic('sunken'),
        line: {
          DEFAULT: semantic('line'),
          strong: semantic('line-strong'),
          soft: semantic('line-soft'),
        },
        ink: {
          DEFAULT: semantic('ink'),
          2: semantic('ink-2'),
          3: semantic('ink-3'),
          inverse: semantic('ink-inverse'),
        },
        brand: {
          DEFAULT: semantic('brand'),
          strong: semantic('brand-strong'),
          soft: semantic('brand-soft'),
          muted: semantic('brand-muted'),
          ink: semantic('brand-ink'),
        },
        primary: {
          DEFAULT: semantic('primary'),
          fg: semantic('primary-fg'),
          hover: semantic('primary-hover'),
        },
        success: { DEFAULT: semantic('success'), soft: semantic('success-soft') },
        warning: { DEFAULT: semantic('warning'), soft: semantic('warning-soft') },
        danger: { DEFAULT: semantic('danger'), soft: semantic('danger-soft') },
        info: { DEFAULT: semantic('info'), soft: semantic('info-soft') },
        accent: {
          DEFAULT: semantic('accent'),
          soft: semantic('accent-soft'),
          ink: semantic('accent-ink'),
        },
      },
      fontFamily: {
        sans: ['Vazirmatn', 'system-ui', '-apple-system', 'Segoe UI', 'sans-serif'],
      },
      fontSize: {
        /* Persian needs a slightly larger body size + looser leading than
           Latin UI text to stay legible at small sizes. */
        '2xs': ['11px', { lineHeight: '1.5', letterSpacing: '0' }],
        xs: ['12px', { lineHeight: '1.6' }],
        sm: ['13.5px', { lineHeight: '1.65' }],
        base: ['15px', { lineHeight: '1.75' }],
        md: ['16.5px', { lineHeight: '1.65' }],
        lg: ['19px', { lineHeight: '1.5' }],
        xl: ['22px', { lineHeight: '1.4' }],
        '2xl': ['26px', { lineHeight: '1.35' }],
        '3xl': ['32px', { lineHeight: '1.3' }],
      },
      borderRadius: {
        xs: 'var(--r-xs)',
        sm: 'var(--r-sm)',
        md: 'var(--r-md)',
        lg: 'var(--r-lg)',
        xl: 'var(--r-xl)',
        '2xl': 'var(--r-2xl)',
        pill: 'var(--r-pill)',
      },
      boxShadow: {
        xs: 'var(--sh-xs)',
        sm: 'var(--sh-sm)',
        md: 'var(--sh-md)',
        lg: 'var(--sh-lg)',
        bar: 'var(--sh-bar)',
        none: 'none',
      },
      spacing: {
        gutter: 'var(--sp-gutter)',
        tabbar: 'var(--tabbar-h)',
        topbar: 'var(--topbar-h)',
      },
      transitionTimingFunction: {
        out: 'var(--ease-out)',
        'in-out': 'var(--ease-in-out)',
      },
      transitionDuration: {
        fast: '120ms',
        base: '200ms',
        slow: '320ms',
      },
      keyframes: {
        'fade-in': { from: { opacity: '0' }, to: { opacity: '1' } },
        'sheet-up': {
          from: { transform: 'translateY(12%)', opacity: '0.4' },
          to: { transform: 'translateY(0)', opacity: '1' },
        },
        'pop-in': {
          from: { transform: 'scale(0.96)', opacity: '0' },
          to: { transform: 'scale(1)', opacity: '1' },
        },
        'rise-in': {
          from: { transform: 'translateY(8px)', opacity: '0' },
          to: { transform: 'translateY(0)', opacity: '1' },
        },
        'toast-in': {
          from: { transform: 'translateY(16px) scale(0.98)', opacity: '0' },
          to: { transform: 'translateY(0) scale(1)', opacity: '1' },
        },
        shimmer: {
          '100%': { transform: 'translateX(-100%)' },
        },
      },
      animation: {
        'fade-in': 'fade-in var(--dur-base) var(--ease-out) both',
        'sheet-up': 'sheet-up var(--dur-slow) var(--ease-out) both',
        'pop-in': 'pop-in var(--dur-base) var(--ease-out) both',
        'rise-in': 'rise-in var(--dur-slow) var(--ease-out) both',
        'toast-in': 'toast-in var(--dur-slow) var(--ease-out) both',
      },
    },
  },
  plugins: [],
} satisfies Config;
