import { Command } from "commander"
import { apikeyCommand } from "./apikey.ts"

export const generateCommand = new Command("generate")
	.description("Generate various resources")
	.addCommand(apikeyCommand)
