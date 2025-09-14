import { createFileRoute, Navigate, Outlet } from "@tanstack/react-router"
import { Authenticated, Unauthenticated } from "convex/react"

export const Route = createFileRoute("/_authenticated")({
	component: AuthenticatedLayout,
})

function AuthenticatedLayout() {
	return (
		<>
			<Authenticated>
				<Outlet />
			</Authenticated>
			<Unauthenticated>
				<Navigate to="/login" />
			</Unauthenticated>
		</>
	)
}
