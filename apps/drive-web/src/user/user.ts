import { type } from "arktype"

export const User = type({
	id: "string",
	displayName: "string",
	email: "string",
})

export type User = typeof User.infer
