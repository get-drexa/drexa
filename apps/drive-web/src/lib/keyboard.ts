import { atom, useSetAtom } from "jotai"
import { useEffect } from "react"

export enum KeyboardModifier {
	Alt = "Alt",
	Control = "Control",
	Meta = "Meta",
	Shift = "Shift",
}

export const keyboardModifierAtom = atom(new Set<KeyboardModifier>())

const addKeyboardModifierAtom = atom(
	null,
	(get, set, modifier: KeyboardModifier) => {
		const activeModifiers = get(keyboardModifierAtom)
		const nextActiveModifiers = new Set(activeModifiers)
		nextActiveModifiers.add(modifier)
		set(keyboardModifierAtom, nextActiveModifiers)
	},
)
const removeKeyboardModifierAtom = atom(
	null,
	(get, set, modifier: KeyboardModifier) => {
		const activeModifiers = get(keyboardModifierAtom)
		const nextActiveModifiers = new Set(activeModifiers)
		nextActiveModifiers.delete(modifier)
		set(keyboardModifierAtom, nextActiveModifiers)
	},
)

export function useKeyboardModifierListener() {
	const addKeyboardModifier = useSetAtom(addKeyboardModifierAtom)
	const removeKeyboardModifier = useSetAtom(removeKeyboardModifierAtom)

	useEffect(() => {
		const handleKeyDown = (event: KeyboardEvent) => {
			switch (event.key) {
				case "Alt":
					addKeyboardModifier(KeyboardModifier.Alt)
					break
				case "Control":
					addKeyboardModifier(KeyboardModifier.Control)
					break
				case "Meta":
					addKeyboardModifier(KeyboardModifier.Meta)
					break
				case "Shift":
					addKeyboardModifier(KeyboardModifier.Shift)
					break
			}
		}
		const handleKeyUp = (event: KeyboardEvent) => {
			switch (event.key) {
				case "Alt":
					removeKeyboardModifier(KeyboardModifier.Alt)
					break
				case "Control":
					removeKeyboardModifier(KeyboardModifier.Control)
					break
				case "Meta":
					removeKeyboardModifier(KeyboardModifier.Meta)
					break
				case "Shift":
					removeKeyboardModifier(KeyboardModifier.Shift)
					break
			}
		}
		window.addEventListener("keydown", handleKeyDown)
		window.addEventListener("keyup", handleKeyUp)
		return () => {
			window.removeEventListener("keydown", handleKeyDown)
			window.removeEventListener("keyup", handleKeyUp)
		}
	}, [addKeyboardModifier, removeKeyboardModifier])
}

export function isControlOrCommandKeyActive(
	keyboardModifiers: Set<KeyboardModifier>,
) {
	return (
		keyboardModifiers.has(KeyboardModifier.Control) ||
		keyboardModifiers.has(KeyboardModifier.Meta)
	)
}
