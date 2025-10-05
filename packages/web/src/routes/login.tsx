import { createFileRoute } from "@tanstack/react-router"
import { Button } from "../components/ui/button"

export const Route = createFileRoute("/login")({
	component: RouteComponent,
})

function RouteComponent() {
	return (
		<div className="flex h-screen w-full items-center justify-center">
			<Button onClick={() => {}}>Login</Button>
		</div>
	)
}
