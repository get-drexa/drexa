import "@/styles/globals.css"
import { ConvexBetterAuthProvider } from "@convex-dev/better-auth/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createRootRoute, Outlet } from "@tanstack/react-router"
import { ConvexReactClient } from "convex/react"
import { toast } from "sonner"
import { Toaster } from "@/components/ui/sonner"
import { formatError } from "@/lib/error"
import { useKeyboardModifierListener } from "@/lib/keyboard"
import { authClient } from "../auth"

export const Route = createRootRoute({
	component: RootLayout,
})

const convexClient = new ConvexReactClient(process.env.BUN_PUBLIC_CONVEX_URL!)
const queryClient = new QueryClient({
	defaultOptions: {
		mutations: {
			onError: (error) => {
				console.log(error)
				toast.error(formatError(error))
			},
			throwOnError: false,
		},
	},
})

function RootLayout() {
	useKeyboardModifierListener()

	return (
		<QueryClientProvider client={queryClient}>
			<ConvexBetterAuthProvider
				client={convexClient}
				authClient={authClient}
			>
				<Outlet />
				<Toaster />
			</ConvexBetterAuthProvider>
		</QueryClientProvider>
	)
}
