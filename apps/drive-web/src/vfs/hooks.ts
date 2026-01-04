import { useAtomValue } from "jotai"
import { currentDriveAtom } from "@/drive/drive"
import { useCurrentOrganization } from "@/organization/context"
import { buildApiUrl } from "@/lib/api"
import type { FileInfo } from "./vfs"

export function useFileUrl(file: FileInfo) {
	const org = useCurrentOrganization()
	const drive = useAtomValue(currentDriveAtom)
	if (!drive) return ""
	return buildApiUrl(
		`/${org.slug}/drives/${drive.id}/files/${file.id}/content`,
	).toString()
}
