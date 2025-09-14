import { toast } from "sonner"
import { Code as ErrorCode, isApplicationError } from "../../convex/model/error"

const ERROR_MESSAGE = {
	[ErrorCode.DirectoryExists]: "Directory already exists",
	[ErrorCode.FileExists]: "File already exists",
	[ErrorCode.Internal]: "Internal application error",
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
