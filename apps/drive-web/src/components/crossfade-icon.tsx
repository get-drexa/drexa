import {
	type ReactNode,
	type Ref,
	useCallback,
	useEffect,
	useImperativeHandle,
	useRef,
	useState,
} from "react"
import { cn } from "@/lib/utils"

type CrossfadeIconProps = {
	from: ReactNode
	to: ReactNode
	active?: boolean
	className?: string
	ref?: Ref<CrossfadeIconHandle>
}

export type CrossfadeIconHandle = {
	trigger: () => void
}

export function CrossfadeIcon({
	from,
	to,
	active = false,
	className,
	ref,
}: CrossfadeIconProps) {
	const [forcedActive, setForcedActive] = useState(false)
	const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

	const clearTimer = useCallback(() => {
		if (timeoutRef.current) {
			clearTimeout(timeoutRef.current)
			timeoutRef.current = null
		}
	}, [])

	useImperativeHandle(
		ref,
		() => ({
			trigger: () => {
				setForcedActive(true)
				clearTimer()
				timeoutRef.current = setTimeout(() => {
					setForcedActive(false)
					timeoutRef.current = null
				}, 3000)
			},
		}),
		[clearTimer],
	)

	useEffect(() => clearTimer, [clearTimer])

	const isActive = active || forcedActive

	return (
		<div className={cn("relative grid place-items-center", className)}>
			<span
				className={cn(
					"col-start-1 row-start-1 grid place-items-center transition-all duration-200 ease-out",
					isActive
						? "opacity-0 scale-50 blur-sm"
						: "opacity-100 scale-100 blur-0",
				)}
			>
				{from}
			</span>
			<span
				className={cn(
					"col-start-1 row-start-1 grid place-items-center transition-all duration-200 ease-out",
					isActive
						? "opacity-100 scale-100 blur-0"
						: "opacity-0 scale-0 blur-sm",
				)}
			>
				{to}
			</span>
		</div>
	)
}
