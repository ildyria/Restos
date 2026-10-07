import { describe, expect, it } from 'vitest'
import { formatDate, formatScore, joinNames, priceLabel, returnVerdict } from '~/composables/format'
import type { Score } from '~~/shared/types'

const score = (name: string, would_return: boolean | null): Score => ({
  user_id: name.length, name, would_return,
  rating_food: null, rating_service: null, rating_ambiance: null, rating_value: null, rating_quiet: null, overall: null
})

describe('returnVerdict', () => {
  it('is empty when nobody answered', () => {
    expect(returnVerdict([])).toBeNull()
    expect(returnVerdict([score('Sam', null), score('Lea', null)])).toBeNull()
  })
  it('speaks for both when only one answered yes', () => {
    expect(returnVerdict([score('Sam', true), score('Lea', null)])).toEqual({ label: 'We would go back', tone: 'all' })
  })
  it('agrees when everyone says yes', () => {
    expect(returnVerdict([score('Sam', true), score('Lea', true)])).toEqual({ label: 'We’d both go back', tone: 'all' })
    expect(returnVerdict([score('A', true), score('B', true), score('C', true)])?.label).toBe('We’d all go back')
  })
  it('names who would go back when opinions differ', () => {
    expect(returnVerdict([score('Sam', true), score('Lea', false)])).toEqual({ label: 'Sam would go back', tone: 'some' })
  })
  it('says no when nobody would', () => {
    expect(returnVerdict([score('Sam', false)])).toEqual({ label: 'Wouldn’t go back', tone: 'none' })
    expect(returnVerdict([score('Sam', false), score('Lea', false)])?.tone).toBe('none')
  })
})

describe('formatting', () => {
  it('joins names', () => {
    expect(joinNames([])).toBe('')
    expect(joinNames(['Sam'])).toBe('Sam')
    expect(joinNames(['Sam', 'Lea'])).toBe('Sam & Lea')
    expect(joinNames(['A', 'B', 'C'])).toBe('A, B & C')
  })
  it('formats scores and prices', () => {
    expect(formatScore(null)).toBe('–')
    expect(formatScore(4.25)).toBe('4.3')
    expect(formatScore(4)).toBe('4.0')
    expect(priceLabel(3)).toBe('€€€')
    expect(priceLabel(null)).toBe('')
  })
  it('reads dates as local calendar days (no timezone shift)', () => {
    expect(formatDate('2026-01-01')).toBe('1 January 2026')
    expect(formatDate('2026-12-31', { month: 'short', year: 'numeric' })).toBe('Dec 2026')
  })
})
