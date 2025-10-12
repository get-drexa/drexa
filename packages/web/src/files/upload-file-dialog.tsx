import type { Doc } from "@fileone/convex/_generated/dataModel"
import { useMutation } from "@tanstack/react-query"
import {
	atom,
	useAtom,
	useAtomValue,
	useSetAtom,
	useStore,
} from "jotai"
import { atomEffect } from "jotai-effect"
import { FilePlus2Icon, UploadCloudIcon, XIcon } from "lucide-react"
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
import { WithAtom } from "@/components/with-atom"
import { clearFileUploadAtom, fileUploadAtomFamily } from "./store"
import useUploadFile from "./use-upload-file"

type UploadFileDialogProps = {
	targetDirectory: Doc<"directories">
	onClose: () => void
}

// Upload file atoms
export type PickedFile = {
	id: string
	file: File
}

export const pickedFilesAtom = atom<PickedFile[]>([])

export function UploadFileDialog({
	targetDirectory,
	onClose,
}: UploadFileDialogProps) {
	const formId = useId()
	const fileInputRef = useRef<HTMLInputElement>(null)
	const setPickedFiles = useSetAtom(pickedFilesAtom)
	const clearFileUpload = useSetAtom(clearFileUploadAtom)
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

	const uploadFile = useUploadFile({
		targetDirectory,
	})
	const { mutate: uploadFiles, isPending: isUploading } = useMutation({
		mutationFn: async (files: PickedFile[]) => {
			const promises = files.map((pickedFile) =>
				uploadFile({
					file: pickedFile.file,
					onStart: () => {
						store.set(fileUploadAtomFamily(pickedFile.id), 0)
					},
					onProgress: (progress) => {
						store.set(fileUploadAtomFamily(pickedFile.id), progress)
					},
				}).then(() => {
					clearFileUpload(pickedFile.id)
				}),
			)
			await Promise.all(promises)
		},
		onSuccess: () => {
			toast.success("All files uploaded successfully")
			onClose()
		},
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

	function startFileUpload() {
		const pickedFiles = store.get(pickedFilesAtom)
		uploadFiles(pickedFiles)
	}

	return (
		<Dialog
			open
			onOpenChange={(open) => {
				if (!open) onClose()
			}}
		>
			<DialogContent className="sm:max-w-2xl">
				<DialogHeader>
					<DialogTitle>
						{targetDirectory.name ? `Upload file to "${targetDirectory.name}"` : "Upload file"}
					</DialogTitle>
					<DialogDescription>
						Drag and drop files here or click to select files
					</DialogDescription>
				</DialogHeader>

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
					<WithAtom atom={pickedFilesAtom}>
						{(pickedFiles) => (
							<>
								{pickedFiles.length > 0 ? (
									<Button
										variant="outline"
										onClick={openFilePicker}
										disabled={isUploading}
									>
										Select more files
									</Button>
								) : null}
								<Button
									onClick={startFileUpload}
									disabled={isUploading}
									loading={isUploading}
								>
									{pickedFiles.length === 0
										? "Upload"
										: `Upload ${pickedFiles.length} files`}
								</Button>
							</>
						)}
					</WithAtom>
				</DialogFooter>
			</DialogContent>
		</Dialog>
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
	const fileUploadAtom = fileUploadAtomFamily(pickedFile.id)
	const fileUpload = useAtomValue(fileUploadAtom)
	console.log("fileUpload", fileUpload)
	const { file, id } = pickedFile
	return (
		<li
			className="pl-3 pr-1 py-0.5 h-8 hover:bg-muted flex justify-between items-center"
			key={id}
		>
			<span>{file.name}</span>
			{fileUpload ? (
				<Progress
					className="max-w-20"
					value={fileUpload.progress * 100}
				/>
			) : (
				<Button
					variant="ghost"
					size="icon"
					onClick={() => onRemove(pickedFile)}
				>
					<XIcon className="size-4" />
				</Button>
			)}
		</li>
	)
}
