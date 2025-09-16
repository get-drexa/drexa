import { api } from "@convex/_generated/api"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useConvexAuth, useMutation as useConvexMutation } from "convex/react"
import { useEffect } from "react"
import { LoadingSpinner } from "@/components/ui/loading-spinner"

export const Route = createFileRoute("/login_/callback")({
	component: RouteComponent,
})

function RouteComponent() {
	const { isLoading: isLoadingConvexAuth, isAuthenticated } = useConvexAuth()
	const { mutate: syncUser, isPending: isSyncingUser } = useMutation({
		mutationFn: useConvexMutation(api.users.syncUser),
		retry: true,
	})
	const navigate = useNavigate()

	useEffect(() => {
		if (!isLoadingConvexAuth && isAuthenticated && !isSyncingUser) {
			console.log({ isLoadingConvexAuth, isAuthenticated, isSyncingUser })
			syncUser(undefined, {
				onSuccess: () => {
					navigate({
						to: "/",
						replace: true,
					})
				},
			})
		}
	}, [
		isLoadingConvexAuth,
		isAuthenticated,
		syncUser,
		isSyncingUser,
		navigate,
	])

	return (
		<div className="flex h-screen w-full items-center justify-center">
			<LoadingSpinner className="size-10" />
		</div>
	)
}
