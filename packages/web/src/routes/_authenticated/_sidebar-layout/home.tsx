import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/_authenticated/_sidebar-layout/home")({
	component: RouteComponent,
})

function RouteComponent() {
	return <div>Hello "/_authenticated/home"!</div>
}
