/**
 * Shared types that can be safely imported by both client and server code.
 * This file should NOT import any server-only dependencies.
 */

import type { Doc } from "../_generated/dataModel"
import type { DirectoryPath } from "./filesystem"

export type DirectoryInfo = Doc<"directories"> & { path: DirectoryPath }
export type DirectoryItem = DirectoryInfo
export type DirectoryItemKind = "directory" | "file"
