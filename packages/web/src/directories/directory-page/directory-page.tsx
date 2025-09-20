import { api } from "@fileone/convex/_generated/api"
import type {
	DirectoryHandle,
	PathComponent,
} from "@fileone/convex/model/filesystem"
import { useMutation } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useMutation as useConvexMutation } from "convex/react"
import { useAtom, useSetAtom } from "jotai"
import {
	ChevronDownIcon,
	Loader2Icon,
	PlusIcon,
	UploadCloudIcon,
} from "lucide-react"
import React, { type ChangeEvent, Fragment, useContext, useRef } from "react"
import { toast } from "sonner"
import { ImagePreviewDialog } from "@/components/image-preview-dialog"
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { DirectoryIcon } from "../../components/icons/directory-icon"
import { TextFileIcon } from "../../components/icons/text-file-icon"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "../../components/ui/breadcrumb"
import { Button } from "../../components/ui/button"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "../../components/ui/tooltip"
import { useFileDrop } from "../../files/use-file-drop"
import { cn } from "../../lib/utils"
import { DirectoryPageContext } from "./context"
import { DirectoryContentTable } from "./directory-content-table"
import { RenameFileDialog } from "./rename-file-dialog"
import { dragInfoAtom, newItemKindAtom, openedFileAtom } from "./state"

export function DirectoryPage() {
	return (
		<>
			<header className="flex py-1 shrink-0 items-center gap-2 border-b px-4 w-full">
				<FilePathBreadcrumb />
				<div className="ml-auto flex flex-row gap-2">
					<NewDirectoryItemDropdown />
					<UploadFileButton />
				</div>
			</header>
			<div className="w-full">
				<DirectoryContentTable />
			</div>
			<RenameFileDialog />
			<PreviewDialog />
		</>
	)
}

function FilePathBreadcrumb() {
	const { rootDirectory, directory } = useContext(DirectoryPageContext)

	const breadcrumbItems: React.ReactNode[] = []
	for (let i = 1; i < directory.path.length - 1; i++) {
		breadcrumbItems.push(
			<Fragment key={directory.path[i]!.handle.id}>
				<BreadcrumbSeparator />
				<FilePathBreadcrumbItem component={directory.path[i]!} />
			</Fragment>,
		)
	}

	return (
		<Breadcrumb>
			<BreadcrumbList>
				{rootDirectory._id === directory._id ? (
					<BreadcrumbItem>
						<BreadcrumbPage>All Files</BreadcrumbPage>
					</BreadcrumbItem>
				) : (
					<FilePathBreadcrumbItem component={directory.path[0]!} />
				)}
				{breadcrumbItems}
				<BreadcrumbSeparator />
				<BreadcrumbItem>
					<BreadcrumbPage>{directory.name}</BreadcrumbPage>{" "}
				</BreadcrumbItem>
			</BreadcrumbList>
		</Breadcrumb>
	)
}

function FilePathBreadcrumbItem({ component }: { component: PathComponent }) {
	const { isDraggedOver, dropHandlers } = useFileDrop({
		item: component.handle as DirectoryHandle,
		dragInfoAtom,
	})

	const dirName = component.name || "All Files"

	return (
		<Tooltip open={isDraggedOver}>
			<TooltipTrigger asChild>
				<BreadcrumbItem
					className={cn({ "bg-muted": isDraggedOver })}
					{...dropHandlers}
				>
					<BreadcrumbLink asChild>
						<Link to={`/directories/${component.handle.id}`}>
							{dirName}
						</Link>
					</BreadcrumbLink>
				</BreadcrumbItem>
			</TooltipTrigger>
			<TooltipContent>Move to {dirName}</TooltipContent>
		</Tooltip>
	)
}

// tags: upload, uploadfile, uploadfilebutton, fileupload, fileuploadbutton
function UploadFileButton() {
	const { directory } = useContext(DirectoryPageContext)
	const generateUploadUrl = useConvexMutation(api.files.generateUploadUrl)
	const saveFile = useConvexMutation(api.files.saveFile)
	const { mutate: uploadFile, isPending: isUploading } = useMutation({
		mutationFn: async (file: File) => {
			const uploadUrl = await generateUploadUrl()
			const uploadResult = await fetch(uploadUrl, {
				method: "POST",
				body: file,
				headers: {
					"Content-Type": file.type,
				},
			})
			const { storageId } = await uploadResult.json()

			await saveFile({
				storageId,
				name: file.name,
				size: file.size,
				mimeType: file.type,
				directoryId: directory._id,
			})
		},
		onSuccess: () => {
			toast.success("File uploaded successfully.")
		},
	})

	const fileInputRef = useRef<HTMLInputElement>(null)

	const handleClick = () => {
		fileInputRef.current?.click()
	}

	const onFileUpload = async (e: ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0]
		if (file) {
			uploadFile(file)
		}
	}

	return (
		<>
			<input
				hidden
				onChange={onFileUpload}
				ref={fileInputRef}
				type="file"
				name="files"
			/>
			<Button
				size="sm"
				type="button"
				onClick={handleClick}
				disabled={isUploading}
			>
				{isUploading ? (
					<Loader2Icon className="animate-spin size-4" />
				) : (
					<UploadCloudIcon className="size-4" />
				)}
				{isUploading ? "Uploading" : "Upload File"}
			</Button>
		</>
	)
}

function NewDirectoryItemDropdown() {
	const setNewItemKind = useSetAtom(newItemKindAtom)

	const addNewDirectory = () => {
		setNewItemKind("directory")
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
			<DropdownMenuContent>
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

function PreviewDialog() {
	const [openedFile, setOpenedFile] = useAtom(openedFileAtom)

	if (!openedFile) return null

	console.log("openedFile", openedFile)

	switch (openedFile.mimeType) {
		case "image/jpeg":
		case "image/png":
		case "image/gif":
			return (
				<ImagePreviewDialog
					file={openedFile}
					onClose={() => setOpenedFile(null)}
				/>
			)
		default:
			return null
	}
}
