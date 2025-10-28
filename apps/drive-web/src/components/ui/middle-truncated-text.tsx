function MiddleTruncatedText({ children }: { children: string }) {
	const LAST_PART_LENGTH = 3
	const lastPart = children.slice(children.length - LAST_PART_LENGTH)
	const firstPart = children.slice(0, children.length - LAST_PART_LENGTH)
	return (
		<p className="max-w-full flex">
			<span className="flex-1 truncate">{firstPart}</span>
			<span className="w-min">{lastPart}</span>
		</p>
	)
}

export { MiddleTruncatedText }
