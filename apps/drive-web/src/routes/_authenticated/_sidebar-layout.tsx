import { createFileRoute, Outlet } from "@tanstack/react-router"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { DashboardSidebar } from "@/dashboard/dashboard-sidebar"

export const Route = createFileRoute("/_authenticated/_sidebar-layout")({
	component: RouteComponent,
})

function RouteComponent() {
	return (
		<SidebarProvider>
			<div className="flex h-screen w-full">
				<DashboardSidebar />
				<SidebarInset className="overflow-hidden">
					<Outlet />
				</SidebarInset>
			</div>
		</SidebarProvider>
	)
}
