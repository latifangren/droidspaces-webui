/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{svelte,js,ts}'],
  theme: {
    extend: {
      colors: {
        bg: 'var(--bg)',
        paper: 'var(--paper)',
        'panel-alt': 'var(--panel-alt)',
        line: 'var(--line)',
        ink: 'var(--ink)',
        muted: 'var(--muted)',
        primary: 'var(--primary)',
        'primary-text': 'var(--primary-text)',
        secondary: 'var(--secondary)',
        accent: 'var(--accent)',
        lime: 'var(--lime)',
        pink: 'var(--pink)',
        cyan: 'var(--cyan)',
        yellow: 'var(--yellow)',
        purple: 'var(--purple)',
        orange: 'var(--orange)',
      },
      boxShadow: {
        brutal: '4px 5px 0 var(--line)',
        'brutal-sm': '3px 3px 0 var(--line)',
        'brutal-lg': '6px 7px 0 var(--line)',
      },
      borderWidth: {
        brutal: '2px',
      },
    },
  },
  plugins: [],
};
