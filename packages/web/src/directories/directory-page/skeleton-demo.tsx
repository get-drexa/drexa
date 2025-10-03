import { useState } from "react"
import { Button } from "@/components/ui/button"
import { DirectoryContentTableSkeleton } from "./directory-content-table-skeleton"
import { DirectoryPageSkeleton } from "./directory-page-skeleton"

export function SkeletonDemo() {
	const [showPageSkeleton, setShowPageSkeleton] = useState(false)
	const [showTableSkeleton, setShowTableSkeleton] = useState(false)

	return (
		<div className="p-4 space-y-4">
			<div className="flex gap-2">
				<Button
					onClick={() => setShowPageSkeleton(!showPageSkeleton)}
					variant={showPageSkeleton ? "default" : "outline"}
				>
					{showPageSkeleton ? "Hide" : "Show"} Page Skeleton
				</Button>
				<Button
					onClick={() => setShowTableSkeleton(!showTableSkeleton)}
					variant={showTableSkeleton ? "default" : "outline"}
				>
					{showTableSkeleton ? "Hide" : "Show"} Table Skeleton
				</Button>
			</div>

			{showPageSkeleton && (
				<div className="border rounded-lg p-4">
					<h3 className="text-lg font-semibold mb-4">
						Directory Page Skeleton
					</h3>
					<DirectoryPageSkeleton />
				</div>
			)}

			{showTableSkeleton && (
				<div className="border rounded-lg p-4">
					<h3 className="text-lg font-semibold mb-4">
						Directory Content Table Skeleton
					</h3>
					<DirectoryContentTableSkeleton rows={5} />
				</div>
			)}
		</div>
	)
}
