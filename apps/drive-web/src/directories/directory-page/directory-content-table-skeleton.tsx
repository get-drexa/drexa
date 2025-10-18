import { Skeleton } from "@/components/ui/skeleton"
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table"

export function DirectoryContentTableSkeleton({ rows = 8 }: { rows?: number }) {
	return (
		<div className="overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow className="px-4">
						<TableHead
							className="first:pl-4 last:pr-4"
							style={{ width: 24 }}
						>
							<Skeleton className="size-4 rounded" />
						</TableHead>
						<TableHead
							className="first:pl-4 last:pr-4"
							style={{ width: 1000 }}
						>
							<Skeleton className="h-4 w-12" />
						</TableHead>
						<TableHead className="first:pl-4 last:pr-4">
							<Skeleton className="h-4 w-8" />
						</TableHead>
						<TableHead className="first:pl-4 last:pr-4">
							<Skeleton className="h-4 w-20" />
						</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{Array.from({ length: rows }).map((_, index) => (
						// biome-ignore lint/suspicious/noArrayIndexKey: this is static so ok
						<DirectoryContentTableRowSkeleton key={index} />
					))}
				</TableBody>
			</Table>
		</div>
	)
}

function DirectoryContentTableRowSkeleton() {
	return (
		<TableRow>
			<TableCell className="first:pl-4 last:pr-4" style={{ width: 24 }}>
				<Skeleton className="size-4 rounded" />
			</TableCell>
			<TableCell className="first:pl-4 last:pr-4" style={{ width: 1000 }}>
				<div className="flex w-full items-center gap-2">
					<Skeleton className="size-4 rounded" />
					<Skeleton className="h-4 w-32" />
				</div>
			</TableCell>
			<TableCell className="first:pl-4 last:pr-4">
				<Skeleton className="h-4 w-16" />
			</TableCell>
			<TableCell className="first:pl-4 last:pr-4">
				<Skeleton className="h-4 w-24" />
			</TableCell>
		</TableRow>
	)
}
