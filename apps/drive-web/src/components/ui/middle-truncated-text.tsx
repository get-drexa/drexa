import { cn } from "@/lib/utils"

function MiddleTruncatedText({
	children,
	className,
}: {
	children: string
	className?: string
}) {
	const LAST_PART_LENGTH = 3
	const lastPart = children.slice(children.length - LAST_PART_LENGTH)
	const firstPart = children.slice(0, children.length - LAST_PART_LENGTH)
	return (
		<p className={cn("max-w-full flex", className)}>
			<span className="flex-1 truncate">{firstPart}</span>
			<span className="w-min">{lastPart}</span>
		</p>
	)
}

export { MiddleTruncatedText }
