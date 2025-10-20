#!/usr/bin/env bun

import { Command } from "commander"
import { generateCommand } from "./commands/generate/index.ts"

const program = new Command()

program
	.name("drexa")
	.description("Drexa CLI - Admin tools for managing Drexa resources")
	.version("0.1.0")

// Register command groups
program.addCommand(generateCommand)

// Parse command line arguments
program.parse()
