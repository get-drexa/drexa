export const PATH_SEPARATOR = "/"

export function baseName(path: string): string {
	return path.split(PATH_SEPARATOR).pop() ?? ""
}

export function joinPath(...paths: string[]): string {
	return paths.join(PATH_SEPARATOR)
}
