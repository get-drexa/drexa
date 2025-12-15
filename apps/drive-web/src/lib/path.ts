import { type } from "arktype"

export const PathSegment = type({
	name: "string",
	id: "string",
})
export type PathSegment = typeof PathSegment.infer

export const Path = type([PathSegment])
export type Path = typeof Path.infer
