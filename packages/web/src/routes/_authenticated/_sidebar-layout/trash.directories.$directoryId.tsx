import { api } from "@fileone/convex/_generated/api"
import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import {
	type FileSystemItem,
	newFileSystemHandle,
} from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import {
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { ShredderIcon, TrashIcon, UndoIcon } from "lucide-react"
import { useCallback, useEffect } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import {
	Dialog,
	DialogClose,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import { WithAtom } from "@/components/with-atom"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import { DirectoryContentTable } from "@/directories/directory-page/directory-content-table"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { FilePathBreadcrumb } from "@/directories/directory-page/file-path-breadcrumb"
import { FilePreviewDialog } from "@/files/file-preview-dialog"
import type { FileDragInfo } from "@/files/use-file-drop"
import { backgroundTaskProgressAtom } from "../../../dashboard/state"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/trash/directories/$directoryId",
)({
	component: RouteComponent,
})

const contextMenuTargetItemsAtom = atom<FileSystemItem[]>([])
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const isDeleteConfirmationDialogOpenAtom = atom(false)
const openedFileAtom = atom<Doc<"files"> | null>(null)
const optimisticRemovedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)

function RouteComponent() {
	const { directoryId } = Route.useParams()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)
	const directory = useConvexQuery(api.files.fetchDirectory, {
		directoryId,
	})
	const directoryContent = useConvexQuery(
		api.filesystem.fetchDirectoryContent,
		{
			directoryId,
			trashed: true,
		},
	)
	const setContextMenuTargetItems = useSetAtom(contextMenuTargetItemsAtom)
	const setOpenedFile = useSetAtom(openedFileAtom)

	const directoryUrlFn = useCallback(
		(directory: Doc<"directories">) =>
			`/trash/directories/${directory._id}`,
		[],
	)

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	const handleContextMenuRequest = (
		row: Row<FileSystemItem>,
		table: Table<FileSystemItem>,
	) => {
		if (row.getIsSelected()) {
			setContextMenuTargetItems(
				table.getSelectedRowModel().rows.map((row) => row.original),
			)
		} else {
			setContextMenuTargetItems([row.original])
		}
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<FilePathBreadcrumb rootLabel="Trash" />
				<div className="ml-auto flex flex-row gap-2">
					<EmptyTrashButton />
				</div>
			</header>

			<TableContextMenu>
				<div className="w-full">
					<DirectoryContentTable
						filterFn={() => true}
						directoryUrlFn={directoryUrlFn}
						fileDragInfoAtom={fileDragInfoAtom}
						onContextMenu={handleContextMenuRequest}
						onOpenFile={setOpenedFile}
					/>
				</div>
			</TableContextMenu>

			<DeleteConfirmationDialog />

			<WithAtom atom={openedFileAtom}>
				{(openedFile, setOpenedFile) => {
					if (!openedFile) return null
					return (
						<FilePreviewDialog
							file={openedFile}
							onClose={() => setOpenedFile(null)}
						/>
					)
				}}
			</WithAtom>
		</DirectoryPageContext>
	)
}

function TableContextMenu({ children }: React.PropsWithChildren) {
	const setIsDeleteConfirmationDialogOpen = useSetAtom(
		isDeleteConfirmationDialogOpenAtom,
	)

	return (
		<ContextMenu>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			<ContextMenuContent>
				<RestoreContextMenuItem />
				<ContextMenuItem
					variant="destructive"
					onClick={() => {
						setIsDeleteConfirmationDialogOpen(true)
					}}
				>
					<ShredderIcon />
					Delete permanently
				</ContextMenuItem>
			</ContextMenuContent>
		</ContextMenu>
	)
}

function RestoreContextMenuItem() {
	const store = useStore()
	const setOptimisticRemovedItems = useSetAtom(optimisticRemovedItemsAtom)
	const restoreItemsMutation = useConvexMutation(api.filesystem.restoreItems)
	const { mutate: restoreItems, isPending: isRestoring } = useMutation({
		mutationFn: restoreItemsMutation,
		onMutate: ({ handles }) => {
			setOptimisticRemovedItems(
				new Set(handles.map((handle) => handle.id)),
			)
		},
		onSuccess: ({ restored, errors }) => {
			if (errors.length === 0) {
				if (restored.files > 0 && restored.directories > 0) {
					toast.success(
						`Restored ${restored.files} files and ${restored.directories} directories`,
					)
				} else if (restored.files > 0) {
					toast.success(`Restored ${restored.files} files`)
				} else if (restored.directories > 0) {
					toast.success(
						`Restored ${restored.directories} directories`,
					)
				}
			} else {
				toast.warning(
					`Restored ${restored.files} files and ${restored.directories} directories; failed to restore ${errors.length} items`,
				)
			}
		},
	})
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)

	useEffect(() => {
		if (isRestoring) {
			setBackgroundTaskProgress({
				label: "Restoring items…",
			})
		} else {
			setBackgroundTaskProgress(null)
		}
	}, [isRestoring, setBackgroundTaskProgress])

	const onClick = () => {
		const targetItems = store.get(contextMenuTargetItemsAtom)
		restoreItems({
			handles: targetItems.map(newFileSystemHandle),
		})
	}

	return (
		<ContextMenuItem onClick={onClick}>
			<UndoIcon />
			Restore
		</ContextMenuItem>
	)
}

function EmptyTrashButton() {
	const setIsDeleteConfirmationDialogOpen = useSetAtom(
		isDeleteConfirmationDialogOpenAtom,
	)

	return (
		<Button
			size="sm"
			type="button"
			variant="destructive"
			onClick={() => {
				setIsDeleteConfirmationDialogOpen(true)
			}}
		>
			<TrashIcon className="size-4" />
			Empty Trash
		</Button>
	)
}

function DeleteConfirmationDialog() {
	const [targetItems, setTargetItems] = useAtom(contextMenuTargetItemsAtom)
	const [isDeleteConfirmationDialogOpen, setIsDeleteConfirmationDialogOpen] =
		useAtom(isDeleteConfirmationDialogOpenAtom)
	const setOptimisticRemovedItems = useSetAtom(optimisticRemovedItemsAtom)

	const deletePermanentlyMutation = useConvexMutation(
		api.filesystem.permanentlyDeleteItems,
	)
	const { mutate: deletePermanently, isPending: isDeleting } = useMutation({
		mutationFn: deletePermanentlyMutation,
		onMutate: ({ handles }) => {
			setOptimisticRemovedItems(
				(prev) =>
					new Set([...prev, ...handles.map((handle) => handle.id)]),
			)
		},
		onSuccess: ({ deleted, errors }, { handles }) => {
			setOptimisticRemovedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
			if (errors.length === 0) {
				toast.success(
					`Deleted ${deleted.files} files and ${deleted.directories} directories`,
				)
			} else {
				toast.warning(
					`Deleted ${deleted.files} files and ${deleted.directories} directories; failed to delete ${errors.length} items`,
				)
			}
			setIsDeleteConfirmationDialogOpen(false)
			setTargetItems([])
		},
	})

	const confirmDelete = () => {
		deletePermanently({
			handles: targetItems.map(newFileSystemHandle),
		})
	}

	return (
		<Dialog
			open={isDeleteConfirmationDialogOpen}
			onOpenChange={setIsDeleteConfirmationDialogOpen}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>
						Permanently delete {targetItems.length} items?
					</DialogTitle>
				</DialogHeader>

				<p>
					{targetItems.length} items will be permanently deleted. They
					will be IRRECOVERABLE.
				</p>

				<DialogFooter>
					<DialogClose asChild>
						<Button variant="outline" disabled={isDeleting}>
							Go back
						</Button>
					</DialogClose>
					<Button
						variant="destructive"
						onClick={confirmDelete}
						disabled={isDeleting}
						loading={isDeleting}
					>
						Yes, delete
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	)
}
