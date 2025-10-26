import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	server: {
    port: 3577, // Change this to your desired port
  },
	plugins: [tailwindcss(), sveltekit()]
});
