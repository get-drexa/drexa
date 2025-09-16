import { createFileRoute } from "@tanstack/react-router"
import { FilesPage } from "@/files/files-page"

export const Route = createFileRoute("/_authenticated/_sidebar-layout/files")({
	component: FilesPage,
})
