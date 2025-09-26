import { type PrimitiveAtom, useAtom } from "jotai"

export function WithAtom<Value>({
	atom,
	children,
}: {
	atom: PrimitiveAtom<Value>
	children: (
		value: Value,
		setValue: (value: Value) => void,
	) => React.ReactNode
}) {
	const [value, setValue] = useAtom(atom)
	return children(value, setValue)
}
