import { type } from "arktype"
import { atom } from "jotai"

export const Account = type({
	id: "string",
	orgId: "string",
	userId: "string",
	role: "'admin'|'member'",
	status: "'invited'|'active'|'suspended'",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Account = typeof Account.infer

export const currentAccountAtom = atom<Account | null>(null)
