import { createFileRoute, Navigate, Outlet } from "@tanstack/react-router"
import { useAtomValue } from "jotai"
import { atomEffect } from "jotai-effect"
import { atomWithQuery } from "jotai-tanstack-query"
import { accountsQuery } from "@/account/api"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { currentAccountAtom } from "../account/account"

export const Route = createFileRoute("/_authenticated")({
	component: AuthenticatedLayout,
})

const accountsAtom = atomWithQuery(() => accountsQuery)
const selectFirstAccountEffect = atomEffect((get, set) => {
	const { data: accounts } = get(accountsAtom)
	const firstAccount = accounts?.[0]
	if (firstAccount && get.peek(currentAccountAtom) === null) {
		set(currentAccountAtom, firstAccount)
	}
})

function AuthenticatedLayout() {
	const { data: accounts, isLoading: isLoadingAccounts } =
		useAtomValue(accountsAtom)

	useAtomValue(selectFirstAccountEffect)

	if (isLoadingAccounts) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	if (!accounts) {
		return <Navigate replace to="/login" />
	}

	return <Outlet />
}
