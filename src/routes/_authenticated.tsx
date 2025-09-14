import {
	createFileRoute,
	Navigate,
	Outlet,
	useLocation,
} from "@tanstack/react-router"
import { useAuth } from "@workos-inc/authkit-react"
import {
	Authenticated,
	AuthLoading,
	Unauthenticated,
	useConvexAuth,
} from "convex/react"
import { useEffect, useState } from "react"
import { LoadingSpinner } from "@/components/ui/loading-spinner"

export const Route = createFileRoute("/_authenticated")({
	component: AuthenticatedLayout,
})

function AuthenticatedLayout() {
	const { search } = useLocation()
	const { isLoading } = useConvexAuth()
	const { isLoading: authKitLoading } = useAuth()
	const [hasProcessedAuth, setHasProcessedAuth] = useState(false)

	// Check if we're in the middle of processing an auth code
	const hasAuthCode = search && typeof search === "object" && "code" in search

	// Track when auth processing is complete
	useEffect(() => {
		if (!authKitLoading && !isLoading) {
			// Delay to ensure auth state is fully synchronized
			const timer = setTimeout(() => {
				setHasProcessedAuth(true)
			}, 500)
			return () => clearTimeout(timer)
		}
	}, [authKitLoading, isLoading])

	// Show loading during auth code processing or while auth state is syncing
	if (hasAuthCode || authKitLoading || isLoading || !hasProcessedAuth) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	return (
		<>
			<Authenticated>
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
