export const PATH_SEPARATOR = "/"

export function baseName(path: string): string {
	return path.split(PATH_SEPARATOR).pop() ?? ""
}

export function isPathAbsolute(path: string): boolean {
	return path.startsWith(PATH_SEPARATOR)
}

export function joinPath(...paths: string[]): string {
	return paths.join(PATH_SEPARATOR)
}

export function splitPath(path: string): string[] {
	const parts = path.split(PATH_SEPARATOR)
	return isPathAbsolute(path) ? parts.slice(1) : parts
}
