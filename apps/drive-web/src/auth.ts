import {
	convexClient,
	crossDomainClient,
} from "@convex-dev/better-auth/client/plugins"
import { createAuthClient } from "better-auth/react"
import { createContext, useContext } from "react"

export type AuthErrorCode = keyof typeof authClient.$ERROR_CODES

export class BetterAuthError extends Error {
	constructor(public readonly errorCode: AuthErrorCode) {
		super(`better-auth error: ${errorCode}`)
	}
}

export const authClient = createAuthClient({
	baseURL: import.meta.env.VITE_CONVEX_SITE_URL,
	plugins: [convexClient(), crossDomainClient()],
})

export type Session = NonNullable<
	Awaited<ReturnType<typeof authClient.useSession>>["data"]
>

export const SessionContext = createContext<Session | null>(null)

export function useSession() {
	const context = useContext(SessionContext)
	if (!context) {
		throw new Error("useSession must be used within a SessionProvider")
	}
	return context
}
