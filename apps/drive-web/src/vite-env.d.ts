/// <reference types="vite/client" />

declare global {
	interface ImportMetaEnv {
		readonly VITE_CONVEX_URL: string
		readonly VITE_CONVEX_SITE_URL: string
	}
}

export {}
