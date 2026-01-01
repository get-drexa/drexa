import { mutationOptions } from "@tanstack/react-query"
import { atom, useAtom, useAtomValue, useSetAtom, useStore } from "jotai"
import { atomEffect } from "jotai-effect"
import { atomWithMutation } from "jotai-tanstack-query"
import {
	CircleAlertIcon,
	CircleCheckIcon,
	FilePlus2Icon,
	UploadCloudIcon,
	XIcon,
} from "lucide-react"
import { nanoid } from "nanoid"
import type React from "react"
import { useId, useMemo, useRef, useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip"
import { formatError } from "@/lib/error"
import { directoryContentQueryKey } from "@/vfs/api"
import type { DirectoryInfoWithPath } from "@/vfs/vfs"
import { currentDriveAtom } from "@/drive/drive"
import {
	clearAllFileUploadStatusesAtom,
	clearFileUploadStatusesAtom,
	FileUploadStatusKind,
	fileUploadCountAtom,
	fileUploadStatusAtomFamily,
	fileUploadStatusesAtom,
	hasFileUploadsErrorAtom,
	successfulFileUploadCountAtom,
} from "./store"
import { uploadFile } from "./upload"

type UploadFileDialogProps = {
	targetDirectory: DirectoryInfoWithPath
	onClose: () => void
}

// Upload file atoms
export type PickedFile = {
	id: string
	file: File
}

export const pickedFilesAtom = atom<PickedFile[]>([])

function useUploadFilesAtom({
	targetDirectory,
}: {
	targetDirectory: DirectoryInfoWithPath
}) {
	const store = useStore()

	const options = useMemo(
		() =>
			mutationOptions({
				mutationFn: async (files: PickedFile[]) => {
					const drive = store.get(currentDriveAtom)
					if (!drive) throw new Error("No drive selected")

					const promises = files.map((pickedFile) =>
						uploadFile({
							drive,
							file: pickedFile.file,
							targetDirectory,
							onStart: () => {
								store.set(
									fileUploadStatusAtomFamily(pickedFile.id),
									{
										kind: FileUploadStatusKind.InProgress,
										progress: 0,
									},
								)
							},
							onProgress: (progress) => {
								store.set(
									fileUploadStatusAtomFamily(pickedFile.id),
									{
										kind: FileUploadStatusKind.InProgress,
										progress,
									},
								)
							},
						}).catch((error) => {
							store.set(
								fileUploadStatusAtomFamily(pickedFile.id),
								{
									kind: FileUploadStatusKind.Error,
									error,
								},
							)
							throw error
						}),
					)
					return await Promise.allSettled(promises)
				},
				onSuccess: (results, files, _result, { client }) => {
					const remainingPickedFiles: PickedFile[] = []
					results.forEach((result, i) => {
						// biome-ignore lint/style/noNonNullAssertion: results lenght must match input files array length
						const pickedFile = files[i]!
						const statusAtom = fileUploadStatusAtomFamily(
							pickedFile.id,
						)
						switch (result.status) {
							case "fulfilled":
								store.set(statusAtom, {
									kind: FileUploadStatusKind.Success,
								})
								break
							case "rejected":
								store.set(statusAtom, {
									kind: FileUploadStatusKind.Error,
									error: result.reason,
								})
								remainingPickedFiles.push(pickedFile)
								break
						}
					})

					if (remainingPickedFiles.length === 0) {
						toast.success("All files uploaded successfully")
					}

					// Invalidate all queries for the target directory (with any params)
					const drive = store.get(currentDriveAtom)
					if (drive) {
						client.invalidateQueries({
							queryKey: directoryContentQueryKey(
								drive.id,
								targetDirectory.id,
							),
						})
					}
				},
				onError: (error) => {
					console.error(error)
					toast.error(formatError(error))
				},
			}),
		[store, targetDirectory],
	)

	return useMemo(() => atomWithMutation(() => options), [options])
}
type UploadFilesAtom = ReturnType<typeof useUploadFilesAtom>

export function UploadFileDialog({
	targetDirectory,
	onClose,
}: UploadFileDialogProps) {
	const formId = useId()
	const fileInputRef = useRef<HTMLInputElement>(null)
	const setPickedFiles = useSetAtom(pickedFilesAtom)
	const clearFileUploadStatuses = useSetAtom(clearFileUploadStatusesAtom)
	const store = useStore()

	const updateFileInputEffect = useMemo(
		() =>
			atomEffect((get) => {
				const dataTransfer = new DataTransfer()
				const pickedFiles = get(pickedFilesAtom)
				for (const { file } of pickedFiles) {
					dataTransfer.items.add(file)
				}
				if (fileInputRef.current) {
					fileInputRef.current.files = dataTransfer.files
				}
			}),
		[],
	)
	useAtom(updateFileInputEffect)

	const uploadFilesAtom = useUploadFilesAtom({
		targetDirectory,
	})

	function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
		event.preventDefault()
	}

	function openFilePicker() {
		fileInputRef.current?.click()
	}

	function handleFileChange(event: React.ChangeEvent<HTMLInputElement>) {
		const files = event.target.files
		if (files) {
			setPickedFiles((prev) => [
				...prev,
				...Array.from(files).map((file) => ({ id: nanoid(), file })),
			])
		}
	}

	function onUploadButtonClick() {
		const uploadStatuses = store.get(fileUploadStatusesAtom)
		const fileUploadCount = store.get(fileUploadCountAtom)
		const pickedFiles = store.get(pickedFilesAtom)
		const { mutate: uploadFiles, reset: restUploadFilesMutation } =
			store.get(uploadFilesAtom)

		if (pickedFiles.length === 0) {
			// no files are picked, nothing to upload
			return
		}

		if (fileUploadCount === 0) {
			// no files are being uploaded, upload all picked files
			uploadFiles(pickedFiles)
			return
		}

		const successfulUploads: PickedFile["id"][] = []
		const nextPickedFiles: PickedFile[] = []
		for (const file of pickedFiles) {
			const uploadStatus = uploadStatuses[file.id]
			if (uploadStatus) {
				switch (uploadStatus.kind) {
					case FileUploadStatusKind.Success:
						successfulUploads.push(file.id)
						continue
					case FileUploadStatusKind.InProgress:
						continue
					case FileUploadStatusKind.Error:
						nextPickedFiles.push(file)
						break
				}
			}
		}

		clearFileUploadStatuses(successfulUploads)

		if (successfulUploads.length === pickedFiles.length) {
			// all files were successfully uploaded, close the dialog
			onClose()
		} else {
			// some files were not successfully uploaded, set the next picked files
			setPickedFiles(nextPickedFiles)
			restUploadFilesMutation()
			uploadFiles(nextPickedFiles)
		}
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) onClose()
			}}
		>
			<DialogContent className="sm:max-w-2xl">
				<UploadDialogHeader
					uploadFilesAtom={uploadFilesAtom}
					targetDirectory={targetDirectory}
				/>

				<form id={formId} onSubmit={handleSubmit}>
					<input
						hidden
						multiple
						type="file"
						name="files"
						ref={fileInputRef}
						onChange={handleFileChange}
					/>

					<UploadFileDropContainer>
						<UploadFileArea onClick={openFilePicker} />
					</UploadFileDropContainer>
				</form>

				<DialogFooter>
					<ContinueUploadAfterSuccessfulUploadButton
						uploadFilesAtom={uploadFilesAtom}
					/>
					<SelectMoreFilesButton
						onClick={openFilePicker}
						uploadFilesAtom={uploadFilesAtom}
					/>
					<ClearUploadErrorsButton />
					<UploadButton
						uploadFilesAtom={uploadFilesAtom}
						onClick={onUploadButtonClick}
					/>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	)
}

function UploadDialogHeader({
	uploadFilesAtom,
	targetDirectory,
}: {
	uploadFilesAtom: UploadFilesAtom
	targetDirectory: DirectoryInfoWithPath
}) {
	const { data: uploadResults, isPending: isUploading } =
		useAtomValue(uploadFilesAtom)
	const successfulUploadCount = useAtomValue(successfulFileUploadCountAtom)

	let dialogTitle: string
	let dialogDescription: string
	if (isUploading) {
		dialogTitle = "Uploading files"
		dialogDescription =
			"You can close the dialog while they are being uploaded in the background."
	} else if (
		uploadResults &&
		uploadResults.length > 0 &&
		successfulUploadCount === uploadResults.length
	) {
		dialogTitle = "Files uploaded"
		dialogDescription =
			"Click 'Done' to close the dialog, or select more files to upload."
	} else if (targetDirectory.name) {
		dialogTitle = `Upload file to "${targetDirectory.name}"`
		dialogDescription = "Drag and drop files here or click to select files"
	} else {
		dialogTitle = "Upload file"
		dialogDescription = "Drag and drop files here or click to select files"
	}

	return (
		<DialogHeader>
			<DialogTitle>{dialogTitle}</DialogTitle>
			<DialogDescription>{dialogDescription}</DialogDescription>
		</DialogHeader>
	)
}

function ContinueUploadAfterSuccessfulUploadButton({
	uploadFilesAtom,
}: {
	uploadFilesAtom: UploadFilesAtom
}) {
	const setPickedFiles = useSetAtom(pickedFilesAtom)
	const clearAllFileUploadStatuses = useSetAtom(
		clearAllFileUploadStatusesAtom,
	)
	const {
		data: uploadResults,
		isPending: isUploading,
		reset: resetUploadFilesMutation,
	} = useAtomValue(uploadFilesAtom)
	const successfulUploadCount = useAtomValue(successfulFileUploadCountAtom)

	if (
		!uploadResults ||
		uploadResults.length === 0 ||
		successfulUploadCount !== uploadResults.length
	) {
		return null
	}

	function resetUploadState() {
		setPickedFiles([])
		clearAllFileUploadStatuses()
		resetUploadFilesMutation()
	}

	return (
		<Button
			variant="outline"
			onClick={resetUploadState}
			disabled={isUploading}
		>
			Upload more files
		</Button>
	)
}

/**
 * allows the user to select more files after they have selected some files for upload. only visible before any upload has been started.
 */
function SelectMoreFilesButton({
	onClick,
	uploadFilesAtom,
}: {
	onClick: () => void
	uploadFilesAtom: UploadFilesAtom
}) {
	const pickedFiles = useAtomValue(pickedFilesAtom)
	const fileUploadCount = useAtomValue(fileUploadCountAtom)
	const { isPending: isUploading } = useAtomValue(uploadFilesAtom)

	if (pickedFiles.length === 0 || fileUploadCount > 0) {
		return null
	}

	return (
		<Button variant="outline" onClick={onClick} disabled={isUploading}>
			Select more files
		</Button>
	)
}

function ClearUploadErrorsButton() {
	const hasUploadErrors = useAtomValue(hasFileUploadsErrorAtom)
	const clearAllFileUploadStatuses = useSetAtom(
		clearAllFileUploadStatusesAtom,
	)
	const setPickedFiles = useSetAtom(pickedFilesAtom)

	if (!hasUploadErrors) {
		return null
	}

	function clearUploadErrors() {
		setPickedFiles([])
		clearAllFileUploadStatuses()
	}

	return (
		<Button variant="outline" onClick={clearUploadErrors}>
			Clear uploads
		</Button>
	)
}

function UploadButton({
	uploadFilesAtom,
	onClick,
}: {
	uploadFilesAtom: UploadFilesAtom
	onClick: () => void
}) {
	const pickedFiles = useAtomValue(pickedFilesAtom)
	const hasUploadErrors = useAtomValue(hasFileUploadsErrorAtom)
	const fileUploadCount = useAtomValue(fileUploadCountAtom)
	const { isPending: isUploading } = useAtomValue(uploadFilesAtom)

	let label: string
	if (hasUploadErrors) {
		label = "Retry failed uploads"
	} else if (pickedFiles.length > 0) {
		if (fileUploadCount > 0) {
			label = "Done"
		} else {
			label = `Upload ${pickedFiles.length} files`
		}
	} else {
		label = "Upload"
	}

	return (
		<Button onClick={onClick} disabled={isUploading} loading={isUploading}>
			{label}
		</Button>
	)
}

function UploadFileDropContainer({ children }: React.PropsWithChildren) {
	const [draggedFiles, setDraggedFiles] = useState<DataTransferItem[]>([])
	const setPickedFiles = useSetAtom(pickedFilesAtom)

	function handleDragOver(e: React.DragEvent) {
		e.preventDefault()
		const items = Array.from(e.dataTransfer.items)
		const draggedFiles = []
		for (const item of items) {
			if (item.kind === "file") {
				draggedFiles.push(item)
			}
		}
		setDraggedFiles(draggedFiles)
	}

	function handleDragLeave() {
		setDraggedFiles([])
	}

	function handleDrop(e: React.DragEvent) {
		e.preventDefault()
		const items = Array.from(e.dataTransfer.items)
		const droppedFiles: PickedFile[] = []
		for (const item of items) {
			const file = item.getAsFile()
			if (file) {
				droppedFiles.push({
					id: nanoid(),
					file,
				})
			}
		}
		setPickedFiles((prev) => [...prev, ...droppedFiles])
		setDraggedFiles([])
	}

	return (
		<section
			onDragOver={handleDragOver}
			onDragLeave={handleDragLeave}
			onDrop={handleDrop}
			aria-label="File drop area"
			className="relative"
		>
			{children}
			{draggedFiles.length > 0 ? (
				<div className="border border-accent bg-primary text-primary-foreground absolute inset-0 rounded flex flex-col items-center justify-center text-sm space-y-1">
					<FilePlus2Icon className="animate-bounce" />
					<p>Drop {draggedFiles.length} files here</p>
				</div>
			) : null}
		</section>
	)
}

// tag: uploadfilearea area fileuploadarea
function UploadFileArea({ onClick }: { onClick: () => void }) {
	const [pickedFiles, setPickedFiles] = useAtom(pickedFilesAtom)

	function removeSelectedFile(file: PickedFile) {
		setPickedFiles((prev) => prev.filter((f) => f.id !== file.id))
	}

	if (pickedFiles.length > 0) {
		return (
			<PickedFilesList
				pickedFiles={pickedFiles}
				onRemoveFile={removeSelectedFile}
			/>
		)
	}

	return (
		<button
			type="button"
			className="w-full h-48 border-2 rounded border-dashed border-border flex flex-col items-center justify-center text-muted-foreground text-sm space-y-1 hover:bg-muted transition-all hover:border-solid"
			onClick={onClick}
		>
			<UploadCloudIcon />
			<span>Click to select files or drag and drop them here</span>
		</button>
	)
}

function PickedFilesList({
	pickedFiles,
	onRemoveFile,
}: {
	pickedFiles: PickedFile[]
	onRemoveFile: (file: PickedFile) => void
}) {
	return (
		<ul className="min-h-48 border border-border rounded bg-card text-sm">
			{pickedFiles.map((file: PickedFile) => (
				<PickedFileItem
					key={file.id}
					file={file}
					onRemove={onRemoveFile}
				/>
			))}
		</ul>
	)
}

function PickedFileItem({
	file: pickedFile,
	onRemove,
}: {
	file: PickedFile
	onRemove: (file: PickedFile) => void
}) {
	const fileUploadAtom = fileUploadStatusAtomFamily(pickedFile.id)
	const fileUpload = useAtomValue(fileUploadAtom)
	const { file, id } = pickedFile

	let statusIndicator: React.ReactNode
	if (!fileUpload) {
		statusIndicator = (
			<Button
				variant="ghost"
				size="icon"
				onClick={() => onRemove(pickedFile)}
			>
				<XIcon className="size-4" />
			</Button>
		)
	} else {
		switch (fileUpload.kind) {
			case FileUploadStatusKind.InProgress:
				statusIndicator = (
					<Progress
						className="max-w-20"
						value={fileUpload.progress * 100}
					/>
				)
				break
			case FileUploadStatusKind.Error:
				statusIndicator = (
					<Tooltip>
						<TooltipTrigger>
							<CircleAlertIcon className="pr-2 text-destructive" />
						</TooltipTrigger>
						<TooltipContent>
							<p>
								Failed to upload file:{" "}
								{formatError(fileUpload.error)}
							</p>
						</TooltipContent>
					</Tooltip>
				)
				break
			case FileUploadStatusKind.Success:
				statusIndicator = (
					<Tooltip>
						<TooltipTrigger>
							<CircleCheckIcon className="pr-2 text-green-500" />
						</TooltipTrigger>
						<TooltipContent>
							<p>File uploaded</p>
						</TooltipContent>
					</Tooltip>
				)
				break
		}
	}

	return (
		<li
			className="pl-3 pr-1 py-0.5 h-8 hover:bg-muted flex justify-between items-center border-b border-border"
			key={id}
		>
			<p>{file.name} </p>
			{statusIndicator}
		</li>
	)
}
