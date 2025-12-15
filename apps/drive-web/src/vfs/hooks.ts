import { useAtomValue } from "jotai"
import { useEffect } from "react"
import { fileUrlAtom } from "./api"
import type { FileInfo } from "./vfs"

export function useFileUrl(file: FileInfo) {
	const fileUrl = useAtomValue(fileUrlAtom(file.id))
	useEffect(
		() => () => {
			fileUrlAtom.remove(file.id)
		},
		[file.id],
	)
	return fileUrl
}
