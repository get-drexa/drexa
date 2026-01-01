import { createFileRoute, Navigate, Outlet } from "@tanstack/react-router"
import { useAtomValue } from "jotai"
import { atomEffect } from "jotai-effect"
import { atomWithQuery } from "jotai-tanstack-query"
import { LoadingSpinner } from "@/components/ui/loading-spinner"
import { drivesQuery } from "@/drive/api"
import { currentDriveAtom } from "@/drive/drive"

export const Route = createFileRoute("/_authenticated")({
	component: AuthenticatedLayout,
})

const drivesAtom = atomWithQuery(() => drivesQuery)
const selectFirstDriveEffect = atomEffect((get, set) => {
	const { data: drives } = get(drivesAtom)
	const firstDrive = drives?.[0]
	if (firstDrive && get.peek(currentDriveAtom) === null) {
		set(currentDriveAtom, firstDrive)
	}
})

function AuthenticatedLayout() {
	const { data: drives, isLoading: isLoadingDrives } =
		useAtomValue(drivesAtom)

	useAtomValue(selectFirstDriveEffect)

	if (isLoadingDrives) {
		return (
			<div className="flex h-screen w-full items-center justify-center">
				<LoadingSpinner className="size-10" />
			</div>
		)
	}

	if (!drives) {
		return <Navigate replace to="/login" />
	}

	return <Outlet />
}
