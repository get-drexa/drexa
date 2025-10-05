import {
	createFileRoute,
	Navigate,
	Outlet,
	useLocation,
} from "@tanstack/react-router"
import {
	Authenticated,
	AuthLoading,
	Unauthenticated,
	useConvexAuth,
} from "convex/react"
import { useEffect, useState } from "react"
import { authClient, SessionContext } from "@/auth"
import { LoadingSpinner } from "@/components/ui/loading-spinner"

export const Route = createFileRoute("/_authenticated")({
	component: AuthenticatedLayout,
})

function AuthenticatedLayout() {
	const { search } = useLocation()
	const { isLoading } = useConvexAuth()
	const { data: session, isPending: sessionLoading } = authClient.useSession()
	const [hasProcessedAuth, setHasProcessedAuth] = useState(false)

	// Check if we're in the middle of processing an auth code
	const hasAuthCode = search && typeof search === "object" && "code" in search

	// Track when auth processing is complete
	useEffect(() => {
		if (!sessionLoading && !isLoading) {
			// Delay to ensure auth state is fully synchronized
			const timer = setTimeout(() => {
				setHasProcessedAuth(true)
			}, 500)
			return () => clearTimeout(timer)
		}
	}, [sessionLoading, isLoading])

	// Show loading during auth code processing or while auth state is syncing
	if (hasAuthCode || sessionLoading || isLoading || !hasProcessedAuth) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	return (
		<>
			<Authenticated>
				{session ? (
					<SessionContext value={session}>
						<Outlet />
					</SessionContext>
				) : null}
				<Outlet />
			</Authenticated>
			<Unauthenticated>
				<Navigate replace to="/login" />
			</Unauthenticated>
			<AuthLoading>
				<div className="flex h-screen w-full items-center justify-center">
					<LoadingSpinner className="size-10" />
				</div>
			</AuthLoading>
		</>
	)
}
