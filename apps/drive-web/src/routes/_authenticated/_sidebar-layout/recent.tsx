import { api } from "@fileone/convex/api"
import type { Doc } from "@fileone/convex/dataModel"
import { newFileHandle } from "@fileone/convex/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute, Link } from "@tanstack/react-router"
import {
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom, useAtomValue, useSetAtom } from "jotai"
import { FolderInputIcon, TrashIcon } from "lucide-react"
import { useCallback } from "react"
import { toast } from "sonner"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { backgroundTaskProgressAtom } from "@/dashboard/state"
import type { FileGridSelection } from "@/files/file-grid"
import { FileGrid } from "@/files/file-grid"
import { formatError } from "@/lib/error"

export const Route = createFileRoute("/_authenticated/_sidebar-layout/recent")({
	component: RouteComponent,
})

const selectedFilesAtom = atom(new Set() as FileGridSelection)
const contextMenuTargetItem = atom<Doc<"files"> | null>(null)

function RouteComponent() {
	return (
		<main className="p-4">
			<RecentFilesContextMenu>
				<RecentFilesGrid />
			</RecentFilesContextMenu>
		</main>
	)
}

function RecentFilesGrid() {
	const recentFiles = useConvexQuery(api.filesystem.fetchRecentFiles, {
		limit: 100,
	})
	const [selectedFiles, setSelectedFiles] = useAtom(selectedFilesAtom)
	const setContextMenuTargetItem = useSetAtom(contextMenuTargetItem)

	const handleContextMenu = useCallback(
		(file: Doc<"files">, _event: React.MouseEvent) => {
			setContextMenuTargetItem(file)
		},
		[setContextMenuTargetItem],
	)

	return (
		<FileGrid
			files={recentFiles ?? []}
			selectedFiles={selectedFiles}
			onSelectionChange={setSelectedFiles}
			onContextMenu={handleContextMenu}
		/>
	)
}

function RecentFilesContextMenu({ children }: { children: React.ReactNode }) {
	const targetItem = useAtomValue(contextMenuTargetItem)
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)

	const { mutate: moveToTrash } = useMutation({
		mutationFn: useConvexMutation(api.filesystem.moveToTrash),
		onMutate: () => {
			setBackgroundTaskProgress({
				label: "Moving to trash…",
			})
		},
		onSuccess: () => {
			setBackgroundTaskProgress(null)
			toast.success("Moved to trash")
		},
		onError: (error) => {
			toast.error("Failed to move to trash", {
				description: formatError(error),
			})
		},
	})

	return (
		<ContextMenu>
			<ContextMenuTrigger asChild>
				<div>{children}</div>
			</ContextMenuTrigger>
			{targetItem && (
				<ContextMenuContent>
					<ContextMenuItem>
						<Link
							to={`/directories/${targetItem.directoryId}`}
							className="flex flex-row items-center gap-2"
						>
							<FolderInputIcon />
							Open in directory
						</Link>
					</ContextMenuItem>
					<ContextMenuItem
						variant="destructive"
						onClick={() => {
							moveToTrash({
								handles: [newFileHandle(targetItem._id)],
							})
						}}
					>
						<TrashIcon />
						Move to trash
					</ContextMenuItem>
				</ContextMenuContent>
			)}
		</ContextMenu>
	)
}
