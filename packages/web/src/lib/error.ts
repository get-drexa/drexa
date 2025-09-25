import {
	Code as ErrorCode,
	isApplicationError,
} from "@fileone/convex/model/error"
import { toast } from "sonner"

const ERROR_MESSAGE = {
	[ErrorCode.DirectoryExists]: "Directory already exists",
	[ErrorCode.FileExists]: "File already exists",
	[ErrorCode.Internal]: "Internal application error",
	[ErrorCode.Conflict]: "Conflict",
	[ErrorCode.DirectoryNotFound]: "Directory not found",
	[ErrorCode.FileNotFound]: "File not found",
	[ErrorCode.Unauthenticated]: "Unauthenticated",
} as const

export function formatError(error: unknown): string {
	if (isApplicationError(error)) {
		return ERROR_MESSAGE[error.data.code]
	}
	if (error instanceof Error) {
		return error.message
	}
	return "Unknown error"
}

export function defaultOnError(error: unknown) {
	console.log(error)
	toast.error(formatError(error))
}

export function withDefaultOnError(fn: (error: unknown) => void) {
	return (error: unknown) => {
		defaultOnError(error)
		fn(error)
	}
}
