import * as readline from "node:readline/promises"
import chalk from "chalk"

function createReadlineInterface() {
	return readline.createInterface({
		input: process.stdin,
		output: process.stdout,
	})
}

export async function promptText(message: string): Promise<string> {
	const rl = createReadlineInterface()
	try {
		const input = await rl.question(chalk.cyan(`${message} `))

		if (!input || input.trim() === "") {
			console.error(chalk.red("✗ Input is required"))
			process.exit(1)
		}
		return input.trim()
	} finally {
		rl.close()
	}
}

export async function promptNumber(
	message: string,
	defaultValue?: number,
): Promise<number> {
	const rl = createReadlineInterface()
	try {
		const defaultStr = defaultValue
			? chalk.dim(` (default: ${defaultValue})`)
			: ""
		const input = await rl.question(chalk.cyan(`${message}${defaultStr} `))

		if ((!input || input.trim() === "") && defaultValue !== undefined) {
			return defaultValue
		}

		if (!input || input.trim() === "") {
			console.error(chalk.red("✗ Input is required"))
			process.exit(1)
		}

		const num = Number.parseInt(input.trim(), 10)
		if (Number.isNaN(num) || num <= 0) {
			console.error(chalk.red("✗ Please enter a valid positive number"))
			process.exit(1)
		}

		return num
	} finally {
		rl.close()
	}
}

export async function promptOptionalDate(
	message: string,
): Promise<Date | undefined> {
	const rl = createReadlineInterface()
	try {
		const input = await rl.question(
			chalk.cyan(`${message} `) +
				chalk.dim("(optional, format: YYYY-MM-DD) "),
		)

		if (!input || input.trim() === "") {
			return undefined
		}

		const date = new Date(input.trim())
		if (Number.isNaN(date.getTime())) {
			console.error(
				chalk.red("✗ Invalid date format. Please use YYYY-MM-DD"),
			)
			process.exit(1)
		}

		if (date < new Date()) {
			console.error(chalk.red("✗ Expiration date must be in the future"))
			process.exit(1)
		}

		return date
	} finally {
		rl.close()
	}
}

export async function promptConfirm(
	message: string,
	defaultValue = false,
): Promise<boolean> {
	const rl = createReadlineInterface()
	try {
		const defaultStr = defaultValue
			? chalk.dim(" (Y/n)")
			: chalk.dim(" (y/N)")
		const input = await rl.question(chalk.cyan(`${message}${defaultStr} `))

		if (!input || input.trim() === "") {
			return defaultValue
		}

		const normalized = input.toLowerCase().trim()
		return normalized === "y" || normalized === "yes"
	} finally {
		rl.close()
	}
}
