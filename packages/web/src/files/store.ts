import { atom } from "jotai"
import { atomFamily } from "jotai/utils"

export enum FileUploadStatusKind {
	InProgress = "InProgress",
	Error = "Error",
	Success = "Success",
}

export type FileUploadInProgress = {
	kind: FileUploadStatusKind.InProgress
	progress: number
}

export type FileUploadError = {
	kind: FileUploadStatusKind.Error
	error: unknown
}

export type FileUploadSuccess = {
	kind: FileUploadStatusKind.Success
}

export type FileUploadStatus =
	| FileUploadInProgress
	| FileUploadError
	| FileUploadSuccess

export const fileUploadsAtom = atom<Record<string, FileUploadStatus>>({})

export const fileUploadStatusAtomFamily = atomFamily((id: string) =>
	atom(
		(get) => get(fileUploadsAtom)[id],
		(get, set, status: FileUploadStatus) => {
			const fileUploads = { ...get(fileUploadsAtom) }
			fileUploads[id] = status
			set(fileUploadsAtom, fileUploads)
		},
	),
)

export const clearFileUploadAtom = atom(null, (get, set, id: string) => {
	const fileUploads = { ...get(fileUploadsAtom) }
	delete fileUploads[id]
	fileUploadStatusAtomFamily.remove(id)
	set(fileUploadsAtom, fileUploads)
})

export const clearFileUploadStatusesAtom = atom(
	null,
	(get, set, ids: string[]) => {
		const fileUploads = { ...get(fileUploadsAtom) }
		for (const id of ids) {
			if (fileUploads[id]) {
				delete fileUploads[id]
			}
			fileUploadStatusAtomFamily.remove(id)
		}
		set(fileUploadsAtom, fileUploads)
	},
)

export const fileUploadCountAtom = atom(
	(get) => Object.keys(get(fileUploadsAtom)).length,
)

export const inProgressFileUploadCountAtom = atom((get) => {
	const statuses = get(fileUploadsAtom)
	let count = 0
	for (const status in statuses) {
		if (statuses[status]?.kind === FileUploadStatusKind.InProgress) {
			count += 1
		}
	}
	return count
})

export const successfulFileUploadCountAtom = atom((get) => {
	const statuses = get(fileUploadsAtom)
	let count = 0
	for (const status in statuses) {
		if (statuses[status]?.kind === FileUploadStatusKind.Success) {
			count += 1
		}
	}
	return count
})

export const hasFileUploadsErrorAtom = atom((get) => {
	const statuses = get(fileUploadsAtom)
	for (const status in statuses) {
		if (statuses[status]?.kind === FileUploadStatusKind.Error) {
			return true
		}
	}
	return false
})
