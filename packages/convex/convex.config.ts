import { defineApp } from "convex/server"
import betterAuth from "./betterauth/convex.config"

const app = defineApp()
app.use(betterAuth)

export default app
