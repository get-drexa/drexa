import { atom } from "jotai"
import { atomFamily } from "jotai/utils"

type FileUpload = {
	id: string
	progress: number
}

export const fileUploadsAtom = atom<Record<string, FileUpload>>({})

export const fileUploadAtomFamily = atomFamily((id: string) =>
	atom(
		(get) => get(fileUploadsAtom)[id],
		(get, set, progress: number) => {
			const fileUploads = { ...get(fileUploadsAtom) }
			fileUploads[id] = { id, progress }
			set(fileUploadsAtom, fileUploads)
		},
	),
)

export const clearFileUploadAtom = atom(null, (get, set, id: string) => {
	const fileUploads = { ...get(fileUploadsAtom) }
	delete fileUploads[id]
	fileUploadAtomFamily.remove(id)
	set(fileUploadsAtom, fileUploads)
})

export const fileUploadCountAtom = atom(
	(get) => Object.keys(get(fileUploadsAtom)).length,
)

export const hasFileUploadsAtom = atom((get) => get(fileUploadCountAtom) > 0)
