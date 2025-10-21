import { Hono } from "hono"
import type { ApiKeyContextVariable } from "./auth"
import type { ConvexContextVariables } from "./convex"

type ContextVariables = ConvexContextVariables & ApiKeyContextVariable

export function newRouter() {
	return new Hono<{
		Variables: ContextVariables
	}>()
}
