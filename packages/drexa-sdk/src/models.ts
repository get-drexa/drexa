import { type } from "arktype"

// Runtime definitions are internal. The package entry point exports only their types.
export const User = type({
	id: "string",
	displayName: "string",
	email: "string",
})
export type User = typeof User.infer

export const Account = type({
	id: "string",
	orgId: "string",
	userId: "string",
	role: "'admin' | 'member'",
	status: "'invited' | 'active' | 'suspended'",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Account = typeof Account.infer

export const OrganizationKind = type("'personal' | 'team'")
export type OrganizationKind = typeof OrganizationKind.infer
export const Organization = type({
	id: "string",
	kind: OrganizationKind,
	name: "string",
	slug: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Organization = typeof Organization.infer

export const Drive = type({
	id: "string",
	name: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	storageUsageBytes: "number",
	storageQuotaBytes: "number",
})
export type Drive = typeof Drive.infer
export const DriveListItem = Drive.and({ rootDir: "string" })
export type DriveListItem = typeof DriveListItem.infer
export const DriveInfo = Drive.and({ rootDirId: "string" })
export type DriveInfo = typeof DriveInfo.infer

export const PathSegment = type({ name: "string", id: "string" })
export type PathSegment = typeof PathSegment.infer
export const Path = PathSegment.array()
export type Path = typeof Path.infer

export const FileInfo = type({
	kind: "'file'",
	id: "string",
	"parentId?": "string",
	name: "string",
	size: "number",
	mimeType: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	"deletedAt?": "string.date.iso.parse",
})
export type FileInfo = typeof FileInfo.infer
export const DirectoryInfo = type({
	kind: "'directory'",
	id: "string",
	"parentId?": "string",
	name: "string",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
	"deletedAt?": "string.date.iso.parse",
})
export type DirectoryInfo = typeof DirectoryInfo.infer
export const DirectoryInfoWithPath = DirectoryInfo.and({ path: Path })
export type DirectoryInfoWithPath = typeof DirectoryInfoWithPath.infer
export const DirectoryItem = type.or(DirectoryInfo, FileInfo)
export type DirectoryItem = typeof DirectoryItem.infer
export const DirectoryContent = DirectoryItem.array()
export type DirectoryContent = typeof DirectoryContent.infer
export const DirectoryContentResponse = type({
	items: DirectoryContent,
	"nextCursor?": "string",
})
export type DirectoryContentResponse = typeof DirectoryContentResponse.infer
export const MoveDirectoryItemsResult = type({
	items: DirectoryContent,
	moved: "string[]",
	conflicts: "string[]",
	errors: type({ id: "string", error: "string" }).array(),
})
export type MoveDirectoryItemsResult = typeof MoveDirectoryItemsResult.infer

export const Share = type({
	id: "string",
	expiresAt: "string.date.iso.parse | null",
	createdAt: "string.date.iso.parse",
	updatedAt: "string.date.iso.parse",
})
export type Share = typeof Share.infer
export const UploadStatus = type("'pending' | 'completed' | 'failed'")
export type UploadStatus = typeof UploadStatus.infer
export const Upload = type({
	id: "string",
	status: UploadStatus,
	uploadUrl: "string",
})
export type Upload = typeof Upload.infer

export type TokenDelivery = "cookie" | "body"
export type LoginInput = {
	email: string
	password: string
	tokenDelivery: TokenDelivery
}
export type SignUpInput = LoginInput & { displayName: string }
export const LoginResponse = type({
	user: User,
	organizations: Organization.array(),
	"accessToken?": "string",
	"refreshToken?": "string",
})
export type LoginResponse = typeof LoginResponse.infer
export const SignUpResponse = type({
	user: User,
	account: Account,
	drive: Drive,
	"accessToken?": "string",
	"refreshToken?": "string",
})
export type SignUpResponse = typeof SignUpResponse.infer
export const Tokens = type({ accessToken: "string", refreshToken: "string" })
export type Tokens = typeof Tokens.infer
export type RefreshTokenInput = { refreshToken: string }

export type DriveScope = { orgSlug: string; driveId: string }
export type CreateDirectoryInput = { parentId: string; name: string }
export type CreateUploadInput = CreateDirectoryInput
export type RenameInput = { name: string }
export type MoveItemsInput = { items: string[] }
export type DirectoryContentOrderBy = "name" | "createdAt" | "updatedAt"
export type DirectoryContentOrderDirection = "asc" | "desc"
export type ListDirectoryOptions = {
	orderBy?: DirectoryContentOrderBy
	direction?: DirectoryContentOrderDirection
	limit?: number
	cursor?: string
}
export type ListSharesOptions = { includesExpired?: boolean }
export type UpdateShareInput = {
	/** A Date or ISO 8601 string; null clears the expiry. */
	expiresAt?: Date | string | null
}
export type CreateShareInput = UpdateShareInput & { items: string[] }
