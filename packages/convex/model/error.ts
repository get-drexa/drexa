import { ConvexError } from "convex/values"

export enum Code {
	Conflict = "Conflict",
	DirectoryExists = "DirectoryExists",
	DirectoryNotFound = "DirectoryNotFound",
	FileExists = "FileExists",
	FileNotFound = "FileNotFound",
	Internal = "Internal",
	Unauthenticated = "Unauthenticated",
}

export type ApplicationErrorData = { code: Code; message?: string }
export type ApplicationError = ConvexError<ApplicationErrorData>

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

export function createJson(code: Code, message?: string): ApplicationErrorData {
	return {
		code,
		message:
			code === Code.Internal ? "Internal application error" : message,
	}
}
