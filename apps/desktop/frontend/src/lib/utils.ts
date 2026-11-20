import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const MAX_PATH_CHARS = 100

// Long paths lose their start, not their end: the file name is the part that matters.
export function shortenPath(path: string, max = MAX_PATH_CHARS): string {
  return path.length <= max ? path : '…' + path.slice(path.length - (max - 1))
}
