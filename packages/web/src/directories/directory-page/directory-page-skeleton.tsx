import { Skeleton } from "@/components/ui/skeleton"
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbList,
} from "../../components/ui/breadcrumb"
import { Button } from "../../components/ui/button"
import { DirectoryContentTableSkeleton } from "./directory-content-table-skeleton"

export function DirectoryPageSkeleton() {
	return (
		<>
			<header className="flex py-1 shrink-0 items-center gap-2 border-b px-4 w-full">
				<BreadcrumbSkeleton />
				<div className="ml-auto flex flex-row gap-2">
					<Button size="sm" type="button" variant="outline" disabled>
						<Skeleton className="size-4 rounded" />
						<Skeleton className="h-4 w-8" />
						<Skeleton className="size-4 rounded" />
					</Button>
					<Button size="sm" type="button" disabled>
						<Skeleton className="size-4 rounded" />
						<Skeleton className="h-4 w-16" />
					</Button>
				</div>
			</header>
			<div className="w-full">
				<DirectoryContentTableSkeleton />
			</div>
		</>
	)
}

function BreadcrumbSkeleton() {
	return (
		<Breadcrumb>
			<BreadcrumbList>
				<BreadcrumbItem>
					<Skeleton className="h-4 w-16" />
				</BreadcrumbItem>
			</BreadcrumbList>
		</Breadcrumb>
	)
}
