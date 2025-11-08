import { generateApiKey, newPrefix } from "@drexa/auth"
import chalk from "chalk"
import { Command } from "commander"
import { promptNumber, promptOptionalDate, promptText } from "../../prompts.ts"

export const apikeyCommand = new Command("apikey")
	.description("Generate a new API key")
	.action(async () => {
		console.log(chalk.bold.blue("\n🔑 Generate API Key\n"))

		// Prompt for all required information
		const prefixInput = await promptText(
			"Enter API key prefix (e.g., 'proxy', 'admin'):",
		)
		const prefix = newPrefix(prefixInput)

		if (!prefix) {
			console.error(
				chalk.red(
					'✗ Invalid prefix: cannot contain "-" character. Please use alphanumeric characters only.',
				),
			)
			process.exit(1)
		}

		const keyByteLength = await promptNumber("Enter key byte length:", 32)
		const description = await promptText("Enter description:")
		const expiresAt = await promptOptionalDate("Enter expiration date")

		console.log(chalk.dim("\n⏳ Generating API key...\n"))

		// Generate the API key
		const result = await generateApiKey({
			prefix,
			keyByteLength,
			description,
			expiresAt,
		})

		// Display results
		console.log(chalk.green.bold("✓ API Key Generated Successfully!\n"))
		console.log(chalk.gray("─".repeat(60)))
		console.log(
			chalk.yellow.bold(
				"\n⚠️  IMPORTANT: Save the unhashed key now. It won't be shown again!\n",
			),
		)
		console.log(chalk.bold("Unhashed Key ") + chalk.dim("(save this):"))
		console.log(chalk.green(`  ${result.unhashedKey}\n`))
		console.log(chalk.gray("─".repeat(60)))
		console.log(
			chalk.bold("\nHashed Key ") +
				chalk.dim("(store this in your database):"),
		)
		console.log(chalk.dim(`  ${result.hashedKey}\n`))
		console.log(chalk.bold("Description:"))
		console.log(chalk.white(`  ${result.description}\n`))

		if (result.expiresAt) {
			console.log(chalk.bold("Expires At:"))
			console.log(chalk.yellow(`  ${result.expiresAt.toISOString()}\n`))
		} else {
			console.log(chalk.bold("Expires At:"))
			console.log(chalk.dim("  Never\n"))
		}

		console.log(chalk.gray("─".repeat(60)) + "\n")
	})
