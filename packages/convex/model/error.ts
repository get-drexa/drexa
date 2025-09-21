import { ConvexError } from "convex/values"

export enum Code {
	Conflict = "Conflict",
	DirectoryExists = "DirectoryExists",
	DirectoryNotFound = "DirectoryNotFound",
	FileExists = "FileExists",
	Internal = "Internal",
	Unauthenticated = "Unauthenticated",
}

export type ApplicationError = ConvexError<{ code: Code; message?: string }>

export function isApplicationError(error: unknown): error is ApplicationError {
	return error instanceof ConvexError && "code" in error.data
}

export function create(code: Code, message?: string): ApplicationError {
	return new ConvexError({
		code,
		message:
			code === Code.Internal ? "Internal application error" : message,
	})
}
