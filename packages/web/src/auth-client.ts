import {
	convexClient,
	crossDomainClient,
} from "@convex-dev/better-auth/client/plugins"
import { createAuthClient } from "better-auth/react"

export type AuthErrorCode = keyof typeof authClient.$ERROR_CODES

export const authClient = createAuthClient({
	baseURL: process.env.BUN_PUBLIC_CONVEX_SITE_URL,
	plugins: [convexClient(), crossDomainClient()],
})
