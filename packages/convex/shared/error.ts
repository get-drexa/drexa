import { ConvexError } from "convex/values"

export enum ErrorCode {
	Conflict = "Conflict",
	DirectoryExists = "DirectoryExists",
	FileExists = "FileExists",
	Internal = "Internal",
	Unauthenticated = "Unauthenticated",
	NotFound = "NotFound",
	StorageQuotaExceeded = "StorageQuotaExceeded",
}

export type ApplicationErrorData = { code: ErrorCode; message?: string }

export function isApplicationError(
	error: unknown,
): error is ApplicationErrorData {
	return (
		error !== null &&
		typeof error === "object" &&
		"code" in error &&
		"message" in error &&
		Object.values(ErrorCode).includes(
			(error as { code: string }).code as ErrorCode,
		)
	)
}

export function createErrorData(
	code: ErrorCode,
	message?: string,
): ApplicationErrorData {
	return {
		code,
		message:
			code === ErrorCode.Internal
				? "Internal application error"
				: message,
	}
}

export function error(data: ApplicationErrorData): never {
	throw new ConvexError(data)
}
