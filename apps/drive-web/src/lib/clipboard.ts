import { mutationOptions } from "@tanstack/react-query"

export const copyToClipboardMutation = mutationOptions({
	mutationFn: async (text: string) => {
		if (
			typeof navigator !== "undefined" &&
			navigator.clipboard?.writeText
		) {
			await navigator.clipboard.writeText(text)
			return text
		}

		if (typeof document !== "undefined") {
			const textarea = document.createElement("textarea")
			textarea.value = text
			textarea.setAttribute("readonly", "")
			textarea.style.position = "fixed"
			textarea.style.opacity = "0"
			textarea.style.left = "-9999px"

			document.body.appendChild(textarea)
			textarea.select()

			const succeeded = document.execCommand("copy")
			document.body.removeChild(textarea)

			if (!succeeded) {
				throw new Error("Failed to copy to clipboard")
			}

			return text
		}

		throw new Error("Clipboard is not available")
	},
})
