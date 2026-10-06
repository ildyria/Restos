import type { Meal, Score } from '~~/shared/types'

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

/** "Sam", "Sam & Lea", "Sam, Lea & Kim". */
export function joinNames(names: string[]) {
  if (names.length <= 1) return names[0] ?? ''
  return `${names.slice(0, -1).join(', ')} & ${names.at(-1)}`
}

export type ReturnTone = 'all' | 'some' | 'none'

/** Sums up the would-go-back answers, ignoring people who didn't answer. */
export function returnVerdict(scores: Score[]): { label: string, tone: ReturnTone } | null {
  const answered = scores.filter(s => s.would_return !== null)
  if (!answered.length) return null
  const yes = answered.filter(s => s.would_return)
  if (!yes.length) return { label: 'Wouldn’t go back', tone: 'none' }
  // A lone answer speaks for both of us, so no name.
  if (answered.length === 1) return { label: 'We would go back', tone: 'all' }
  if (yes.length === answered.length) {
    return { label: answered.length === 2 ? 'We’d both go back' : 'We’d all go back', tone: 'all' }
  }
  return { label: `${joinNames(yes.map(s => s.name))} would go back`, tone: 'some' }
}

/** Today as YYYY-MM-DD in local time. */
export const todayISO = () => new Date().toLocaleDateString('en-CA')

/** The meal you're most likely recording right now. */
export const mealNow = (): Meal => new Date().getHours() < 16 ? 'lunch' : 'dinner'
