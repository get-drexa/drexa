import { createFileRoute } from "@tanstack/react-router"
import { useAuth } from "@workos-inc/authkit-react"
import { Button } from "../components/ui/button"

export const Route = createFileRoute("/login")({
	component: RouteComponent,
})

function RouteComponent() {
	const { signIn } = useAuth()

	return <Button onClick={() => signIn()}>Login</Button>
}
