import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [
    svelte()
  ],
  base: './',
  build: {
    outDir: 'dist-svelte',
    emptyOutDir: true,
    rollupOptions: {
      // No externals needed — we use window.require() for Node/Electron modules
      // which Vite leaves untouched as runtime expressions
    }
  },
  resolve: {
    alias: {
      '$lib': '/src/lib',
      '$components': '/src/components'
    }
  }
});
