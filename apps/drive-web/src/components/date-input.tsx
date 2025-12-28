import { CalendarIcon } from "lucide-react"
import * as React from "react"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import { Input } from "@/components/ui/input"
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover"
import { formatDate, isValidDate } from "@/lib/date"
import { cn } from "@/lib/utils"

export type DateInputHandle = {
	readonly date: Date | undefined
}

type DateInputProps = {
	ref?: React.Ref<DateInputHandle>
	value?: Date
	defaultValue?: Date
	onDateChange?: (date: Date | undefined) => void
	placeholder?: string
	inputId?: string
	disabled?: boolean
	startMonth?: Date
	endMonth?: Date
	className?: string
	inputClassName?: string
	buttonClassName?: string
}

export function DateInput(props: DateInputProps) {
	const {
		ref: imperativeRef,
		value,
		defaultValue,
		onDateChange,
		placeholder = "June 01, 2025",
		inputId,
		disabled = false,
		startMonth: startMonthProp,
		endMonth: endMonthProp,
		className,
		inputClassName,
		buttonClassName,
	} = props
	const isControlled = Object.hasOwn(props, "value")
	const [open, setOpen] = React.useState(false)
	const [internalDate, setInternalDate] = React.useState<Date | undefined>(
		isControlled ? value : defaultValue,
	)
	const selectedDate = isControlled ? value : internalDate
	const [month, setMonth] = React.useState<Date | undefined>(selectedDate)
	const [inputValue, setInputValue] = React.useState(formatDate(selectedDate))
	const baseYear = (selectedDate ?? new Date()).getFullYear()
	const startMonth =
		startMonthProp ?? new Date(baseYear - 100, 0, 1)
	const endMonth =
		endMonthProp ?? new Date(baseYear + 100, 11, 1)

	React.useEffect(() => {
		setInputValue(formatDate(selectedDate))
		setMonth(selectedDate)
	}, [selectedDate])

	React.useImperativeHandle(
		imperativeRef,
		() => ({
			get date() {
				return selectedDate
			},
		}),
		[selectedDate],
	)

	const commitDate = React.useCallback(
		(nextDate: Date | undefined) => {
			if (!isControlled) {
				setInternalDate(nextDate)
			}

			onDateChange?.(nextDate)
		},
		[isControlled, onDateChange],
	)

	return (
		<div
			className={cn("relative flex gap-2", className)}
			data-slot="date-input"
		>
			<Input
				id={inputId}
				value={inputValue}
				placeholder={placeholder}
				disabled={disabled}
				className={cn("bg-background pr-10", inputClassName)}
				onChange={(event) => {
					const nextValue = event.target.value
					setInputValue(nextValue)

					if (!nextValue.trim()) {
						commitDate(undefined)
						setMonth(undefined)
					}
				}}
				onBlur={() => {
					const nextDate = new Date(inputValue)

					if (isValidDate(nextDate)) {
						commitDate(nextDate)
						setMonth(nextDate)
						setInputValue(formatDate(nextDate))
					}
				}}
				onKeyDown={(event) => {
					if (event.key === "ArrowDown") {
						event.preventDefault()
						if (!disabled) {
							setOpen(true)
						}
					}

					if (event.key === "Enter") {
						const nextDate = new Date(inputValue)

						if (isValidDate(nextDate)) {
							commitDate(nextDate)
							setMonth(nextDate)
							setInputValue(formatDate(nextDate))
						}
					}
				}}
			/>
			<Popover
				open={open}
				onOpenChange={(nextOpen) => {
					if (disabled) {
						return
					}

					setOpen(nextOpen)
				}}
			>
				<PopoverTrigger asChild>
					<Button
						id={inputId ? `${inputId}-picker` : undefined}
						type="button"
						variant="ghost"
						disabled={disabled}
						className={cn(
							"absolute top-1/2 right-2 size-6 -translate-y-1/2",
							buttonClassName,
						)}
					>
						<CalendarIcon className="size-3.5" />
						<span className="sr-only">Select date</span>
					</Button>
				</PopoverTrigger>
				<PopoverContent
					className="w-auto overflow-hidden p-0"
					align="end"
					alignOffset={-8}
					sideOffset={10}
				>
					<Calendar
						mode="single"
						selected={selectedDate}
						captionLayout="dropdown"
						startMonth={startMonth}
						endMonth={endMonth}
						month={month}
						onMonthChange={setMonth}
						onSelect={(nextDate) => {
							commitDate(nextDate)
							setInputValue(formatDate(nextDate))
							if (nextDate) {
								setMonth(nextDate)
							}
							setOpen(false)
						}}
					/>
					<div className="border-t p-2">
						<Button
							type="button"
							variant="ghost"
							size="sm"
							className="w-full justify-center"
							onClick={() => {
								const today = new Date()
								commitDate(today)
								setInputValue(formatDate(today))
								setMonth(today)
							}}
						>
							Today
						</Button>
					</div>
				</PopoverContent>
			</Popover>
		</div>
	)
}
