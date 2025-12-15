import { type } from "arktype"
import { atom } from "jotai"

export const Account = type({
	id: "string",
	userId: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	storageUsageBytes: "number",
	storageQuotaBytes: "number",
})
export type Account = typeof Account.infer

export const currentAccountAtom = atom<Account | null>(null)
