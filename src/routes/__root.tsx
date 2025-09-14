import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { Toaster } from "@/components/ui/sonner"
import DashboardSidebar from "@/dashboard/dashboard-sidebar"
import "@/styles/globals.css"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Outlet, createRootRoute } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/router-devtools"
import { ConvexProvider, ConvexReactClient } from "convex/react"

export const Route = createRootRoute({
	component: RootLayout,
})

const convexClient = new ConvexReactClient(process.env.BUN_PUBLIC_CONVEX_URL!)
const queryClient = new QueryClient()

function RootLayout() {
	return (
		<QueryClientProvider client={queryClient}>
			<ConvexProvider client={convexClient}>
				<SidebarProvider>
					<div className="flex h-screen w-full">
						<DashboardSidebar />
						<SidebarInset>
							<Outlet />
						</SidebarInset>
					</div>
					<Toaster />
					<TanStackRouterDevtools />
				</SidebarProvider>
			</ConvexProvider>
		</QueryClientProvider>
	)
}
