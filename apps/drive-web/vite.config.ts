import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
	plugins: [
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		react(),
		tailwindcss(),
	],
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	server: {
		port: 3000,
		host: true,
		fs: {
			allow: [".."],
		},
	},
	optimizeDeps: {
		include: ["convex/react", "convex/values", "convex-helpers"],
		// Workaround for better-auth bug: https://github.com/better-auth/better-auth/issues/4457
		// Vite's esbuild incorrectly transpiles better-call dependency causing 'super' keyword errors
		exclude: ["better-auth", "@convex-dev/better-auth"],
		esbuildOptions: {
			target: "esnext",
		},
	},
})
