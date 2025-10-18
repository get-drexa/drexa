import {
	type Atom,
	type ExtractAtomArgs,
	type ExtractAtomResult,
	type ExtractAtomValue,
	type PrimitiveAtom,
	type SetStateAction,
	useAtom,
	type WritableAtom,
} from "jotai"
import type * as React from "react"

type SetAtom<Args extends unknown[], Result> = (...args: Args) => Result

export function WithAtom<Value, Args extends unknown[], Result>(props: {
	atom: WritableAtom<Value, Args, Result>
	children: (
		value: Awaited<Value>,
		setAtom: SetAtom<Args, Result>,
	) => React.ReactNode
}): React.ReactNode
export function WithAtom<Value>(props: {
	atom: PrimitiveAtom<Value>
	children: (
		value: Awaited<Value>,
		setAtom: SetAtom<[SetStateAction<Value>], void>,
	) => React.ReactNode
}): React.ReactNode
export function WithAtom<Value>(props: {
	atom: Atom<Value>
	children: (value: Awaited<Value>, setAtom: never) => React.ReactNode
}): React.ReactNode
export function WithAtom<
	AtomType extends WritableAtom<unknown, never[], unknown>,
>(props: {
	atom: AtomType
	children: (
		value: Awaited<ExtractAtomValue<AtomType>>,
		setAtom: SetAtom<
			ExtractAtomArgs<AtomType>,
			ExtractAtomResult<AtomType>
		>,
	) => React.ReactNode
}): React.ReactNode
export function WithAtom<AtomType extends Atom<unknown>>(props: {
	atom: AtomType
	children: (
		value: Awaited<ExtractAtomValue<AtomType>>,
		setAtom: never,
	) => React.ReactNode
}): React.ReactNode
export function WithAtom<Value, Args extends unknown[], Result>({
	atom,
	children,
}: {
	atom: Atom<Value> | WritableAtom<Value, Args, Result>
	children: (
		value: Awaited<Value>,
		setAtom: SetAtom<Args, Result> | never,
	) => React.ReactNode
}) {
	const [value, setAtom] = useAtom(atom as WritableAtom<Value, Args, Result>)
	return children(value, setAtom)
}
