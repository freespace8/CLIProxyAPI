export const STORAGE_KEY = 'cli-proxy-dashboard-key'

export function readStoredKey(): string {
  return window.localStorage.getItem(STORAGE_KEY) ?? ''
}

export function formatCompactCount(value: number): string {
  const formatter = new Intl.NumberFormat('en', {
    maximumFractionDigits: 1,
    notation: 'compact',
  })
  return formatter.format(value)
}
