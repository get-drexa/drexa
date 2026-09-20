import {
	type CreateDirectoryInput,
	DirectoryContentResponse,
	DirectoryInfo,
	DirectoryInfoWithPath,
	FileInfo,
	type ListDirectoryOptions,
	type ListSharesOptions,
	MoveDirectoryItemsResult,
	type MoveItemsInput,
	type RenameInput,
	Share,
} from "./models"
import { type RequestOptions, segment, type Transport } from "./transport"

// Shared by drive routes and share-link routes; permissions are enforced by the server.
export function createCatalog(transport: Transport, base: string) {
	const url = (
		path: string,
		query?: Record<string, string | number | boolean | undefined>,
	) => transport.url(`${base}${path}`, query)
	const filePath = (id: string) => `/files/${segment(id)}`
	const directoryPath = (id: string) => `/directories/${segment(id)}`
	return {
		files: {
			get: (id: string, options?: RequestOptions) =>
				transport.json(
					"GET",
					url(filePath(id)),
					FileInfo,
					undefined,
					options,
				),
			rename: (
				id: string,
				input: RenameInput,
				options?: RequestOptions,
			) =>
				transport.json(
					"PATCH",
					url(filePath(id)),
					FileInfo,
					input,
					options,
				),
			trash: (id: string, options?: RequestOptions) =>
				transport.json(
					"DELETE",
					url(filePath(id), { trash: true }),
					FileInfo,
					undefined,
					options,
				),
			delete: (id: string, options?: RequestOptions) =>
				transport.empty(
					"DELETE",
					url(filePath(id), { trash: false }),
					options,
				),
			trashMany: (ids: string[], options?: RequestOptions) => {
				if (ids.length === 0) return Promise.resolve([] as FileInfo[])
				return transport.json(
					"DELETE",
					url("/files", { id: ids.join(","), trash: true }),
					FileInfo.array(),
					undefined,
					options,
				)
			},
			deleteMany: (ids: string[], options?: RequestOptions) => {
				if (ids.length === 0) return Promise.resolve()
				return transport.empty(
					"DELETE",
					url("/files", { id: ids.join(","), trash: false }),
					options,
				)
			},
			/** Bare URL for cookie/public consumers. Use download() for bearer authentication. */
			contentUrl: (id: string) => url(`${filePath(id)}/content`),
			download: (id: string, options?: RequestOptions) =>
				transport.send(
					"GET",
					url(`${filePath(id)}/content`),
					undefined,
					options,
				),
			listShares: (
				id: string,
				input: ListSharesOptions = {},
				options?: RequestOptions,
			) =>
				transport.json(
					"GET",
					url(`${filePath(id)}/shares`, input),
					Share.array(),
					undefined,
					options,
				),
		},
		directories: {
			get: (id: string, options?: RequestOptions) =>
				transport.json(
					"GET",
					url(directoryPath(id), { include: "path" }),
					DirectoryInfoWithPath,
					undefined,
					options,
				),
			list: (
				id: string,
				input: ListDirectoryOptions = {},
				options?: RequestOptions,
			) => {
				const { direction, ...query } = input
				return transport.json(
					"GET",
					url(`${directoryPath(id)}/content`, {
						...query,
						dir: direction,
					}),
					DirectoryContentResponse,
					undefined,
					options,
				)
			},
			create: (input: CreateDirectoryInput, options?: RequestOptions) => {
				const { parentId, name } = input
				return transport.json(
					"POST",
					url("/directories", { include: "path" }),
					DirectoryInfoWithPath,
					{ parentID: parentId, name },
					options,
				)
			},
			rename: (
				id: string,
				input: RenameInput,
				options?: RequestOptions,
			) =>
				transport.json(
					"PATCH",
					url(directoryPath(id)),
					DirectoryInfo,
					input,
					options,
				),
			moveItems: (
				id: string,
				input: MoveItemsInput,
				options?: RequestOptions,
			) => {
				if (input.items.length === 0)
					return Promise.resolve({
						items: [],
						moved: [],
						conflicts: [],
						errors: [],
					} as MoveDirectoryItemsResult)
				return transport.json(
					"POST",
					url(`${directoryPath(id)}/content`),
					MoveDirectoryItemsResult,
					input,
					options,
				)
			},
			trash: (id: string, options?: RequestOptions) =>
				transport.json(
					"DELETE",
					url(directoryPath(id), { trash: true }),
					DirectoryInfo,
					undefined,
					options,
				),
			delete: (id: string, options?: RequestOptions) =>
				transport.empty(
					"DELETE",
					url(directoryPath(id), { trash: false }),
					options,
				),
			trashMany: (ids: string[], options?: RequestOptions) => {
				if (ids.length === 0)
					return Promise.resolve([] as DirectoryInfo[])
				return transport.json(
					"DELETE",
					url("/directories", { id: ids.join(","), trash: true }),
					DirectoryInfo.array(),
					undefined,
					options,
				)
			},
			deleteMany: (ids: string[], options?: RequestOptions) => {
				if (ids.length === 0) return Promise.resolve()
				return transport.empty(
					"DELETE",
					url("/directories", { id: ids.join(","), trash: false }),
					options,
				)
			},
			listShares: (
				id: string,
				input: ListSharesOptions = {},
				options?: RequestOptions,
			) =>
				transport.json(
					"GET",
					url(`${directoryPath(id)}/shares`, input),
					Share.array(),
					undefined,
					options,
				),
		},
	}
}

export type CatalogClient = ReturnType<typeof createCatalog>
