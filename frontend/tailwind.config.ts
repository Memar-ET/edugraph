import type { Config } from 'tailwindcss'

// Design system: "The Register" — simplified palette per 2026-08-16 revision.
// Three colors do all the work: one primary (brand/interactive) and two
// functional status colors (success = positive, attention = needs-attention).
// Everything else is clean neutral gray or white.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Primary — deep ink blue, brand/interactive elements only.
        primary: {
          50:  '#EFF2F7',
          100: '#DCE3ED',
          200: '#B9C7DB',
          300: '#8FA5C2',
          400: '#5F7EA0',
          500: '#3D5C80',
          600: '#284769',
          700: '#1D3450',
          800: '#16233F',
          900: '#0F1826',
        },
        // Clean neutral gray — replaces the warm ledger-paper tone.
        gray: {
          50:  '#F9FAFB',
          100: '#F3F4F6',
          200: '#E5E7EB',
          300: '#D1D5DB',
          400: '#9CA3AF',
          500: '#6B7280',
          600: '#4B5563',
          700: '#374151',
          800: '#1F2937',
          900: '#111827',
        },
        // Success — positive status: mastery, on-track, pass.
        success: {
          50:  '#ECFDF5',
          100: '#D1FAE5',
          200: '#A7F3D0',
          300: '#6EE7B7',
          400: '#34D399',
          500: '#10B981',
          600: '#059669',
          700: '#047857',
          800: '#065F46',
          900: '#064E3B',
        },
        // Attention — needs-attention status: gaps, warnings, reteach.
        attention: {
          50:  '#FFFBEB',
          100: '#FEF3C7',
          200: '#FDE68A',
          300: '#FCD34D',
          400: '#FBBF24',
          500: '#F59E0B',
          600: '#D97706',
          700: '#B45309',
          800: '#92400E',
          900: '#78350F',
        },
        // Kept for backward compat — existing pages reference these names.
        seal: {
          50:  '#FBF3E4', 100: '#F5E3C0', 200: '#EACB8A', 300: '#DDB05C',
          400: '#CC9640', 500: '#B8863B', 600: '#96692A', 700: '#744F1F',
          800: '#523717', 900: '#2E1F0D',
        },
        health: {
          50:  '#EDF4EC', 100: '#D3E6D1', 200: '#A6CDA1', 300: '#74AF6E',
          400: '#4C9245', 500: '#2F6844', 600: '#235A38', 700: '#1B472C',
          800: '#133421', 900: '#0B2114',
        },
        alert: {
          50:  '#FBEEEA', 100: '#F3D3C7', 200: '#E5A78E', 300: '#D67A57',
          400: '#BC5333', 500: '#A23E2A', 600: '#832F1F', 700: '#642417',
          800: '#451A10', 900: '#260E08',
        },
      },
      fontFamily: {
        // IBM Plex Sans is the single type family across all roles and scripts.
        sans:    ['"IBM Plex Sans"', '"IBM Plex Sans Ethiopic"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono:    ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
        // display now resolves to IBM Plex Sans (keeps font-display classes working).
        display: ['"IBM Plex Sans"', 'ui-sans-serif', 'sans-serif'],
      },
    },
  },
  plugins: [],
} satisfies Config
