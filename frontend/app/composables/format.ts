export function formatDate(iso: string, opts: Intl.DateTimeFormatOptions = { day: 'numeric', month: 'long', year: 'numeric' }) {
  // Dates are plain YYYY-MM-DD; parse as local to avoid a timezone off-by-one.
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y!, (m ?? 1) - 1, d ?? 1).toLocaleDateString('en-GB', opts)
}

export function formatScore(n: number | null | undefined) {
  return n == null ? '–' : n.toFixed(1)
}

export function priceLabel(level: number | null | undefined) {
  return level ? '€'.repeat(level) : ''
}

export const COURSES = [
  { label: 'Starter', value: 'starter' },
  { label: 'Main', value: 'main' },
  { label: 'Dessert', value: 'dessert' },
  { label: 'Drink', value: 'drink' },
  { label: 'Other', value: 'other' }
]
