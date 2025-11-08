import {
	type ApplicationErrorData,
	ErrorCode,
	isApplicationError,
} from "@fileone/convex/error"
import { ConvexError } from "convex/values"
import { toast } from "sonner"

const ERROR_MESSAGE = {
	[ErrorCode.DirectoryExists]: "Directory already exists",
	[ErrorCode.FileExists]: "File already exists",
	[ErrorCode.Internal]: "Internal application error",
	[ErrorCode.Conflict]: "Conflict",
	[ErrorCode.Unauthenticated]: "Unauthenticated",
	[ErrorCode.NotFound]: "Not found",
	[ErrorCode.StorageQuotaExceeded]: "Storage is full",
} as const

export function isApplicationConvexError(
	error: unknown,
): error is ConvexError<ApplicationErrorData> {
	return error instanceof ConvexError && isApplicationError(error.data)
}

export function formatError(error: unknown): string {
	if (isApplicationConvexError(error)) {
		return ERROR_MESSAGE[error.data.code]
	}
	if (error instanceof Error) {
		return error.message
	}
	return "Unknown error"
}

export function defaultOnError(error: unknown) {
	if (isApplicationConvexError(error)) {
		toast.error(formatError(error))
	} else {
		console.error("Catastrophic error:", error)
		toast.error("An unexpected error occurred")
	}
}

export function withDefaultOnError(fn: (error: unknown) => void) {
	return (error: unknown) => {
		defaultOnError(error)
		fn(error)
	}
}
