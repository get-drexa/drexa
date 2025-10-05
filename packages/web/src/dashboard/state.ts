import { atom } from "jotai"

type BackgroundTaskProgress = {
	label: string
}

export const backgroundTaskProgressAtom = atom<BackgroundTaskProgress | null>(
	null,
)
