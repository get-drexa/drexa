import { type } from "arktype"
import { atom } from "jotai"

export const Drive = type({
	id: "string",
	name: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	rootDirId: "string",
	storageUsageBytes: "number",
	storageQuotaBytes: "number",
})
export type Drive = typeof Drive.infer

export const currentDriveAtom = atom<Drive | null>(null)
