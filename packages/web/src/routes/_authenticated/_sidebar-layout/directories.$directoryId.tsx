import { api } from "@fileone/convex/_generated/api"
import type { Doc, Id } from "@fileone/convex/_generated/dataModel"
import {
	type FileSystemItem,
	FileType,
	newFileSystemHandle,
} from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import type { Row, Table } from "@tanstack/react-table"
import {
	useMutation as useContextMutation,
	useMutation as useConvexMutation,
	useQuery as useConvexQuery,
} from "convex/react"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import {
	ChevronDownIcon,
	Loader2Icon,
	PlusIcon,
	TextCursorInputIcon,
	TrashIcon,
	UploadCloudIcon,
} from "lucide-react"
import { type ChangeEvent, useCallback, useContext, useRef } from "react"
import { toast } from "sonner"
import { DirectoryIcon } from "@/components/icons/directory-icon"
import { TextFileIcon } from "@/components/icons/text-file-icon"
import { Button } from "@/components/ui/button"
import {
	ContextMenu,
	ContextMenuContent,
	ContextMenuItem,
	ContextMenuTrigger,
} from "@/components/ui/context-menu"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { WithAtom } from "@/components/with-atom"
import { DirectoryPageContext } from "@/directories/directory-page/context"
import { DirectoryContentTable } from "@/directories/directory-page/directory-content-table"
import { DirectoryPageSkeleton } from "@/directories/directory-page/directory-page-skeleton"
import { FilePathBreadcrumb } from "@/directories/directory-page/file-path-breadcrumb"
import { NewDirectoryDialog } from "@/directories/directory-page/new-directory-dialog"
import { RenameFileDialog } from "@/directories/directory-page/rename-file-dialog"
import { FilePreviewDialog } from "@/files/file-preview-dialog"
import { UploadFileDialog } from "@/files/upload-file-dialog"
import type { FileDragInfo } from "@/files/use-file-drop"

export const Route = createFileRoute(
	"/_authenticated/_sidebar-layout/directories/$directoryId",
)({
	component: RouteComponent,
})

enum DialogKind {
	NewDirectory = "NewDirectory",
	UploadFile = "UploadFile",
}

type NewDirectoryDialogData = {
	kind: DialogKind.NewDirectory
}

type UploadFileDialogData = {
	kind: DialogKind.UploadFile
	directory: Doc<"directories">
}

type ActiveDialogData = NewDirectoryDialogData | UploadFileDialogData

// MARK: atoms
const contextMenuTargetItemsAtom = atom<FileSystemItem[]>([])
const activeDialogDataAtom = atom<ActiveDialogData | null>(null)
const fileDragInfoAtom = atom<FileDragInfo | null>(null)
const optimisticDeletedItemsAtom = atom(
	new Set<Id<"files"> | Id<"directories">>(),
)
const openedFileAtom = atom<Doc<"files"> | null>(null)
const itemBeingRenamedAtom = atom<{
	originalItem: FileSystemItem
	name: string
} | null>(null)

// MARK: page entry
function RouteComponent() {
	const { directoryId } = Route.useParams()
	const rootDirectory = useConvexQuery(api.files.fetchRootDirectory)
	const directory = useConvexQuery(api.files.fetchDirectory, {
		directoryId,
	})
	const store = useStore()
	const directoryContent = useConvexQuery(
		api.filesystem.fetchDirectoryContent,
		{
			directoryId,
			trashed: false,
		},
	)
	const setOpenedFile = useSetAtom(openedFileAtom)
	const setContextMenuTargetItems = useSetAtom(contextMenuTargetItemsAtom)

	const tableFilter = useCallback(
		(item: FileSystemItem) =>
			store.get(optimisticDeletedItemsAtom).has(item.doc._id),
		[store],
	)

	const openFile = useCallback(
		(file: Doc<"files">) => {
			setOpenedFile(file)
		},
		[setOpenedFile],
	)

	const directoryUrlFn = useCallback(
		(directory: Doc<"directories">) => `/directories/${directory._id}`,
		[],
	)

	const directoryUrlById = useCallback(
		(directoryId: Id<"directories">) => `/directories/${directoryId}`,
		[],
	)

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

	if (!directory || !directoryContent || !rootDirectory) {
		return <DirectoryPageSkeleton />
	}

	return (
		<DirectoryPageContext
			value={{ rootDirectory, directory, directoryContent }}
		>
			<header className="flex py-2 shrink-0 items-center gap-2 border-b px-4 w-full">
				<FilePathBreadcrumb
					rootLabel="All Files"
					directoryUrlFn={directoryUrlById}
				/>
				<div className="ml-auto flex flex-row gap-2">
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>

			{/* DirectoryContentContextMenu must wrap div instead of DirectoryContentTable, otherwise radix will throw "event.preventDefault is not a function" error, idk why */}
			<DirectoryContentContextMenu>
				<div className="w-full">
					<DirectoryContentTable
						filterFn={tableFilter}
						directoryUrlFn={directoryUrlFn}
						fileDragInfoAtom={fileDragInfoAtom}
						onContextMenu={handleContextMenuRequest}
						onOpenFile={openFile}
					/>
				</div>
			</DirectoryContentContextMenu>

			<WithAtom atom={activeDialogDataAtom}>
				{(data, setData) => (
					<>
						<NewDirectoryDialog
							open={data?.kind === DialogKind.NewDirectory}
							directoryId={directory._id}
							onOpenChange={(open) => {
								if (!open) {
									setData(null)
								}
							}}
						/>
						{data?.kind === DialogKind.UploadFile && (
							<UploadFileDialog
								targetDirectory={data.directory}
								onClose={() => setData(null)}
							/>
						)}
					</>
				)}
			</WithAtom>

			<WithAtom atom={itemBeingRenamedAtom}>
				{(itemBeingRenamed, setItemBeingRenamed) => {
					if (!itemBeingRenamed) return null
					return (
						<RenameFileDialog
							item={itemBeingRenamed.originalItem}
							onRenameSuccess={() => {
								toast.success("File renamed successfully")
								setItemBeingRenamed(null)
							}}
							onClose={() => setItemBeingRenamed(null)}
						/>
					)
				}}
			</WithAtom>

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

// ==================================
// MARK: DirectoryContentContextMenu

function DirectoryContentContextMenu({
	children,
}: {
	children: React.ReactNode
}) {
	const store = useStore()
	const [target, setTarget] = useAtom(contextMenuTargetItemsAtom)
	const setOptimisticDeletedItems = useSetAtom(optimisticDeletedItemsAtom)
	const moveToTrashMutation = useContextMutation(api.filesystem.moveToTrash)

	const { mutate: moveToTrash } = useMutation({
		mutationFn: moveToTrashMutation,
		onMutate: ({ handles }) => {
			setOptimisticDeletedItems(
				(prev) =>
					new Set([...prev, ...handles.map((handle) => handle.id)]),
			)
		},
		onSuccess: ({ deleted, errors }, { handles }) => {
			setOptimisticDeletedItems((prev) => {
				const newSet = new Set(prev)
				for (const handle of handles) {
					newSet.delete(handle.id)
				}
				return newSet
			})
			if (errors.length === 0 && deleted.length === handles.length) {
				toast.success(`Moved ${handles.length} items to trash`)
			} else if (errors.length === handles.length) {
				toast.error("Failed to move to trash")
			} else {
				toast.info(
					`Moved ${deleted.length} items to trash; failed to move ${errors.length} items`,
				)
			}
		},
	})

	const handleDelete = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length > 0) {
			moveToTrash({
				handles: selectedItems.map(newFileSystemHandle),
			})
		}
	}

	return (
		<ContextMenu
			onOpenChange={(open) => {
				if (!open) {
					setTarget([])
				}
			}}
		>
			<ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
			{target.length > 0 && (
				<ContextMenuContent>
					<RenameMenuItem />
					<ContextMenuItem
						variant="destructive"
						onClick={handleDelete}
					>
						<TrashIcon />
						Move to trash
					</ContextMenuItem>
				</ContextMenuContent>
			)}
		</ContextMenu>
	)
}

function RenameMenuItem() {
	const store = useStore()
	const target = useAtomValue(contextMenuTargetItemsAtom)
	const setItemBeingRenamed = useSetAtom(itemBeingRenamedAtom)

	const handleRename = () => {
		const selectedItems = store.get(contextMenuTargetItemsAtom)
		if (selectedItems.length === 1) {
			// biome-ignore lint/style/noNonNullAssertion: length is checked
			const selectedItem = selectedItems[0]!
			setItemBeingRenamed({
				originalItem: selectedItem,
				name: selectedItem.doc.name,
			})
		}
	}

	// Only render if exactly one item is selected
	if (target.length !== 1) {
		return null
	}

	return (
		<ContextMenuItem onClick={handleRename}>
			<TextCursorInputIcon />
			Rename
		</ContextMenuItem>
	)
}

// ==================================

// tags: upload, uploadfile, uploadfilebutton, fileupload, fileuploadbutton
function UploadFileButton() {
	const { directory } = useContext(DirectoryPageContext)
	// const uploadFile = useUploadFile({
	// 	targetDirectory: directory,
	// })
	// const { mutate: uploadFile, isPending: isUploading } = useMutation({
	// 	mutationFn: async (file: File) => {
	// 		const uploadUrl = await generateUploadUrl()
	// 		const uploadResult = await fetch(uploadUrl, {
	// 			method: "POST",
	// 			body: file,
	// 			headers: {
	// 				"Content-Type": file.type,
	// 			},
	// 		})
	// 		const { storageId } = await uploadResult.json()

	// 		await saveFile({
	// 			storageId,
	// 			name: file.name,
	// 			size: file.size,
	// 			mimeType: file.type,
	// 			directoryId: directory._id,
	// 		})
	// 	},
	// 	onSuccess: () => {
	// 		toast.success("File uploaded successfully.")
	// 	},
	// })
	const setActiveDialogData = useSetAtom(activeDialogDataAtom)

	const handleClick = () => {
		setActiveDialogData({
			kind: DialogKind.UploadFile,
			directory: directory,
		})
	}

	return (
		<Button size="sm" type="button" onClick={handleClick}>
			Upload files
		</Button>
	)
}

function NewDirectoryItemDropdown() {
	const [activeDialogData, setActiveDialogData] =
		useAtom(activeDialogDataAtom)

	const addNewDirectory = () => {
		setActiveDialogData({
			kind: DialogKind.NewDirectory,
		})
	}

	const handleCloseAutoFocus = (event: Event) => {
		// If we just created a new item, prevent the dropdown from restoring focus to the trigger
		if (activeDialogData) {
			event.preventDefault()
		}
	}

	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<Button size="sm" type="button" variant="outline">
					<PlusIcon className="size-4" />
					New
					<ChevronDownIcon className="pl-1 size-4 shrink-0" />
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent onCloseAutoFocus={handleCloseAutoFocus}>
				<DropdownMenuItem>
					<TextFileIcon />
					Text file
				</DropdownMenuItem>
				<DropdownMenuItem onClick={addNewDirectory}>
					<DirectoryIcon />
					Directory
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	)
}
