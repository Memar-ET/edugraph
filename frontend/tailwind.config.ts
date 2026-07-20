import type { Config } from 'tailwindcss'

// Design system: "The Register" -- see frontend design notes.
//
// This platform turns paper (curricula, exam scripts, gradebooks) into
// structured records that a school, region, and ministry all trust. The
// palette and type system are drawn from that world -- ledger paper, ink,
// and the wax/ink seal a ministry official stamps on an approved record --
// rather than a generic SaaS blue. `gray` and `primary` are overridden
// (not renamed) so every existing page inherits the identity for free.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Ink -- brand/interactive. Deep desaturated blue, like fountain-pen
        // ink on a register page.
        primary: {
          50: '#EFF2F7',
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
        // Warm ledger-paper neutral, replacing the cool default gray so
        // every bg-gray-*/text-gray-* usage inherits the identity.
        gray: {
          50: '#F7F5EF',
          100: '#EFEBE1',
          200: '#E1DACA',
          300: '#C9BFA8',
          400: '#A69C84',
          500: '#847A64',
          600: '#665D4B',
          700: '#4B4436',
          800: '#332E24',
          900: '#211D16',
        },
        // Seal -- the ministry stamp accent. Used sparingly: approvals,
        // certifications, the one warm highlight on an otherwise ink+paper page.
        seal: {
          50: '#FBF3E4',
          100: '#F5E3C0',
          200: '#EACB8A',
          300: '#DDB05C',
          400: '#CC9640',
          500: '#B8863B',
          600: '#96692A',
          700: '#744F1F',
          800: '#523717',
          900: '#2E1F0D',
        },
        // Health -- mastery, coverage, "this is fine" data states.
        health: {
          50: '#EDF4EC',
          100: '#D3E6D1',
          200: '#A6CDA1',
          300: '#74AF6E',
          400: '#4C9245',
          500: '#2F6844',
          600: '#235A38',
          700: '#1B472C',
          800: '#133421',
          900: '#0B2114',
        },
        // Alert -- gaps, root causes, "reteach now" data states.
        alert: {
          50: '#FBEEEA',
          100: '#F3D3C7',
          200: '#E5A78E',
          300: '#D67A57',
          400: '#BC5333',
          500: '#A23E2A',
          600: '#832F1F',
          700: '#642417',
          800: '#451A10',
          900: '#260E08',
        },
      },
      fontFamily: {
        display: ['"Fraunces"', 'ui-serif', 'Georgia', 'serif'],
        sans: ['"IBM Plex Sans"', '"IBM Plex Sans Ethiopic"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      boxShadow: {
        stamp: '0 1px 2px 0 rgb(15 24 38 / 0.06), 0 1px 1px 0 rgb(15 24 38 / 0.04)',
      },
    },
  },
  plugins: [],
} satisfies Config
