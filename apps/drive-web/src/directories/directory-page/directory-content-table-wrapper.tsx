/**
 * Wrapper component that conditionally loads real or mock table.
 * Only available in dev mode - mock table is never included in production builds.
 */

import type { Row, Table as TableType } from "@tanstack/react-table"
import type { PrimitiveAtom } from "jotai"
import { atom, useAtomValue } from "jotai"
import { atomWithStorage } from "jotai/utils"
import { lazy, Suspense } from "react"
import type { FileDragInfo } from "@/files/use-file-drop"
import type { DirectoryInfo, DirectoryItem, FileInfo } from "@/vfs/vfs"
import { DirectoryContentTable } from "./directory-content-table"

// Atom to control mock table usage
// In dev mode: uses atomWithStorage to persist in localStorage
// In production: uses regular atom (always false, tree-shaken)
// This ensures hooks are always called in the same order
const useMockTableAtom = import.meta.env.DEV
	? atomWithStorage<boolean>("drexa:use-mock-directory-table", false)
	: atom<boolean>(false)

// Conditional lazy import - Vite will tree-shake this entire import in production
// because import.meta.env.DEV is evaluated at build time
const MockDirectoryContentTable = import.meta.env.DEV
	? lazy(() =>
			import("./mock-directory-content-table").then((mod) => ({
				default: mod.MockDirectoryContentTable,
			})),
		)
	: null

type DirectoryContentTableWrapperProps = {
	directoryUrlFn: (directory: DirectoryInfo) => string
	fileDragInfoAtom: PrimitiveAtom<FileDragInfo | null>
	onContextMenu: (
		row: Row<DirectoryItem>,
		table: TableType<DirectoryItem>,
	) => void
	onOpenFile: (file: FileInfo) => void
}

export function DirectoryContentTableWrapper(
	props: DirectoryContentTableWrapperProps,
) {
	// Always call the hook - in production the atom always returns false
	const useMock = useAtomValue(useMockTableAtom)

	// In production, MockDirectoryContentTable is null, so this always renders real table
	if (import.meta.env.DEV && useMock && MockDirectoryContentTable) {
		return (
			<Suspense fallback={<div>Loading mock table...</div>}>
				<MockDirectoryContentTable {...props} />
			</Suspense>
		)
	}

	return <DirectoryContentTable {...props} />
}

/**
 * Dev-only: Export the atom for use in toggle components.
 * This is only available in dev mode and will be tree-shaken in production.
 */
export const mockTableAtom = useMockTableAtom
