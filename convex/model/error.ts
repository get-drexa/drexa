import { ConvexError } from "convex/values"

export enum Code {
	DirectoryExists = "DirectoryExists",
	DirectoryNotFound = "DirectoryNotFound",
	FileExists = "FileExists",
	Internal = "Internal",
	Unauthenticated = "Unauthenticated",
}

export type ApplicationError = ConvexError<{ code: Code; message: string }>

export function isApplicationError(error: unknown): error is ApplicationError {
	return error instanceof ConvexError && "code" in error.data
}

export function create(code: Code, message?: string) {
	return new ConvexError({
		code,
		message,
	})
}
