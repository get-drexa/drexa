export function fileShareUrl(shareToken: string) {
	return `${import.meta.env.VITE_FILE_PROXY_URL}/files/${shareToken}`
}
