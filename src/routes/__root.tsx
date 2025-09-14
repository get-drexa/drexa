import "@/styles/globals.css"
import { ConvexProviderWithAuthKit } from "@convex-dev/workos"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createRootRoute, Outlet } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/router-devtools"
import { AuthKitProvider, useAuth } from "@workos-inc/authkit-react"
import { ConvexReactClient } from "convex/react"
import { toast } from "sonner"
import { formatError } from "@/lib/error"

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
	return (
		<QueryClientProvider client={queryClient}>
			<AuthKitProvider
				clientId={process.env.BUN_PUBLIC_WORKOS_CLIENT_ID!}
				redirectUri={process.env.BUN_PUBLIC_WORKOS_REDIRECT_URI!}
			>
				<ConvexProviderWithAuthKit
					client={convexClient}
					useAuth={useAuth}
				>
					<Outlet />
				</ConvexProviderWithAuthKit>
			</AuthKitProvider>
		</QueryClientProvider>
	)
}
