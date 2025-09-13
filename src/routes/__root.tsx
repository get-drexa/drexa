import { createRootRoute, Outlet } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/router-devtools"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { Toaster } from "@/components/ui/sonner"
import DashboardSidebar from "@/dashboard/dashboard-sidebar"
import "@/styles/globals.css"
import { ConvexProvider, ConvexReactClient } from "convex/react"

export const Route = createRootRoute({
	component: RootLayout,
})

const convexClient = new ConvexReactClient(process.env.BUN_PUBLIC_CONVEX_URL!)

function RootLayout() {
	return (
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
	)
}
