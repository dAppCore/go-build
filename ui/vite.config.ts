import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'CoreBuild',
      fileName: 'core-build',
      formats: ['es'],
    },
    outDir: resolve(__dirname, '../pkg/api/ui/dist'),
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'core-build.js',
      },
    },
  },
});
