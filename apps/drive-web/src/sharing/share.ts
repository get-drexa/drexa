import { type } from "arktype"

export const Share = type({
	id: "string",
	expiresAt: "string.date.iso.parse | null",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Share = typeof Share.infer
