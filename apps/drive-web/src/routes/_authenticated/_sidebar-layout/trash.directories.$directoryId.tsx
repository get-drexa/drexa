import { api } from "@fileone/convex/api"
import type { Doc, Id } from "@fileone/convex/dataModel"
import {
	type FileSystemItem,
	FileType,
	newFileSystemHandle,
} from "@fileone/convex/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import {
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom, useSetAtom, useStore } from "jotai"
import { ShredderIcon, TrashIcon, UndoIcon } from "lucide-react"
import { useCallback, useContext } from "react"
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
import { DirectoryPathBreadcrumb } from "@/directories/directory-path-breadcrumb"
import type { FileDragInfo } from "@/files/use-file-drop"
import { backgroundTaskProgressAtom } from "../../../dashboard/state"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/trash/directories/$directoryId",
)({
	component: RouteComponent,
})

enum ActiveDialogKind {
	DeleteConfirmation = "DeleteConfirmation",
	EmptyTrashConfirmation = "EmptyTrashConfirmation",
}

const contextMenuTargetItemsAtom = atom<FileSystemItem[]>([])
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const activeDialogAtom = atom<ActiveDialogKind | null>(null)
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

	const directoryUrlById = useCallback(
		(directoryId: Id<"directories">) => `/trash/directories/${directoryId}`,
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
				<DirectoryPathBreadcrumb
					directory={directory}
					rootLabel="Trash"
					directoryUrlFn={directoryUrlById}
				/>
				<div className="ml-auto flex flex-row gap-2">
					<EmptyTrashButton />
				</div>
			</header>

			<TableContextMenu>
				<div className="w-full">
					<WithAtom atom={optimisticRemovedItemsAtom}>
						{(optimisticRemovedItems) => (
							<DirectoryContentTable
								hiddenItems={optimisticRemovedItems}
								directoryUrlFn={directoryUrlFn}
								fileDragInfoAtom={fileDragInfoAtom}
								onContextMenu={handleContextMenuRequest}
								onOpenFile={setOpenedFile}
							/>
						)}
					</WithAtom>
				</div>
			</TableContextMenu>

			<DeleteConfirmationDialog />
			<EmptyTrashConfirmationDialog />
		</DirectoryPageContext>
	)
}

function TableContextMenu({ children }: React.PropsWithChildren) {
	const setActiveDialog = useSetAtom(activeDialogAtom)

	return (
		<ContextMenu>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			<ContextMenuContent>
				<RestoreContextMenuItem />
				<ContextMenuItem
					variant="destructive"
					onClick={() => {
						setActiveDialog(ActiveDialogKind.DeleteConfirmation)
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

	const { mutate: restoreItems } = useMutation({
		mutationFn: restoreItemsMutation,
		onMutate: ({ handles }) => {
			setBackgroundTaskProgress({
				label: "Restoring items…",
			})
			setOptimisticRemovedItems(
				new Set(handles.map((handle) => handle.id)),
			)
		},
		onSuccess: ({ restored, errors }) => {
			setBackgroundTaskProgress(null)
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
		onError: (_err, { handles }) => {
			setOptimisticRemovedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
		},
	})
	const setBackgroundTaskProgress = useSetAtom(backgroundTaskProgressAtom)

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
	const setActiveDialog = useSetAtom(activeDialogAtom)

	return (
		<Button
			size="sm"
			type="button"
			variant="destructive"
			onClick={() => {
				setActiveDialog(ActiveDialogKind.EmptyTrashConfirmation)
			}}
		>
			<TrashIcon className="size-4" />
			Empty trash
		</Button>
	)
}

function DeleteConfirmationDialog() {
	const { rootDirectory } = useContext(DirectoryPageContext)
	const [activeDialog, setActiveDialog] = useAtom(activeDialogAtom)
	const [targetItems, setTargetItems] = useAtom(contextMenuTargetItemsAtom)
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
			setActiveDialog(null)
			setTargetItems([])
		},
	})

	const onOpenChange = (open: boolean) => {
		if (open) {
			setActiveDialog(ActiveDialogKind.DeleteConfirmation)
		} else {
			setActiveDialog(null)
		}
	}

	const confirmDelete = () => {
		deletePermanently({
			handles:
				targetItems.length > 0
					? targetItems.map(newFileSystemHandle)
					: [
							newFileSystemHandle({
								kind: FileType.Directory,
								doc: rootDirectory,
							}),
						],
		})
	}

	return (
		<Dialog
			open={activeDialog === ActiveDialogKind.DeleteConfirmation}
			onOpenChange={onOpenChange}
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

function EmptyTrashConfirmationDialog() {
	const [activeDialog, setActiveDialog] = useAtom(activeDialogAtom)

	const { mutate: emptyTrash, isPending: isEmptying } = useMutation({
		mutationFn: useConvexMutation(api.filesystem.emptyTrash),
		onSuccess: () => {
			toast.success("Trash emptied successfully")
			setActiveDialog(null)
		},
	})

	function onOpenChange(open: boolean) {
		if (open) {
			setActiveDialog(ActiveDialogKind.EmptyTrashConfirmation)
		} else {
			setActiveDialog(null)
		}
	}

	function confirmEmpty() {
		emptyTrash(undefined)
	}

	return (
		<Dialog
			open={activeDialog === ActiveDialogKind.EmptyTrashConfirmation}
			onOpenChange={onOpenChange}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Empty your trash?</DialogTitle>
				</DialogHeader>

				<p>
					All items in the trash will be permanently deleted. They
					will be IRRECOVERABLE.
				</p>

				<DialogFooter>
					<DialogClose asChild>
						<Button variant="outline" disabled={isEmptying}>
							No, go back
						</Button>
					</DialogClose>
					<Button
						variant="destructive"
						onClick={confirmEmpty}
						disabled={isEmptying}
						loading={isEmptying}
					>
						Yes, empty trash
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	)
}
