import type { Person } from '~~/shared/types'

/** Everyone who has scored a restaurant, shared across the masthead and pages. */
export function usePeople() {
  const { data } = useFetch<Person[]>('/api/people', { key: 'people', default: () => [] })
  const names = computed(() => joinNames((data.value ?? []).map(p => p.name)))
  return { people: data, names }
}
