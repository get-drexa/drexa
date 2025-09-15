import { ConvexError } from "convex/values"

export enum Code {
	DirectoryExists = "DirectoryExists",
	FileExists = "FileExists",
	Internal = "Internal",
	Unauthenticated = "Unauthenticated",
}

export type ApplicationError = ConvexError<{ code: Code; message: string }>

export function isApplicationError(error: unknown): error is ApplicationError {
	return (
		error instanceof ConvexError &&
		"code" in error.data &&
		"message" in error.data
	)
}

export function create(code: Code, message: string = "unknown error") {
	return new ConvexError({
		code,
		message,
	})
}
