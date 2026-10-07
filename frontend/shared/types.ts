export type Meal = 'lunch' | 'dinner'
export type Course = 'starter' | 'main' | 'dessert' | 'drink' | 'other'

/**
 * A photo belongs to a restaurant, usually through a visit; dish_id is what it shows (its caption).
 * The picture taken from the website has no visit and source 'website'.
 */
export interface Photo {
  id: number
  restaurant_id: number
  visit_id: number | null
  dish_id: number | null
  source: 'upload' | 'website'
  key: string
  width: number
  height: number
  position: number
}

export interface Dish {
  id: number
  visit_id: number
  name: string
  course: Course | null
  notes: string
  position: number
  photos: Photo[]
}

export interface Visit {
  id: number
  restaurant_id: number
  visited_on: string
  meal: Meal
  notes: string
  dishes: Dish[]
  /** Photos without a dish; captioned ones are under their dish. */
  photos: Photo[]
}

export interface VisitInput {
  visited_on: string
  meal: Meal
  notes: string
}

export interface Score {
  user_id: number
  name: string
  rating_food: number | null
  rating_service: number | null
  rating_ambiance: number | null
  rating_value: number | null
  rating_quiet: number | null
  overall: number | null
  would_return: boolean | null
}

export type ScoreInput = Omit<Score, 'name' | 'overall'>

export interface Person {
  id: number
  name: string
}

export interface Restaurant {
  id: number
  slug: string
  name: string
  cuisine: string
  address: string
  city: string
  website: string
  price_level: number | null
  michelin_stars: number | null
  rating_food: number | null
  rating_service: number | null
  rating_ambiance: number | null
  rating_value: number | null
  rating_quiet: number | null
  overall: number | null
  latitude: number | null
  longitude: number | null
  review: string
  rank_position: number | null
  cover_photo_id: number | null
  created_at: string
  updated_at: string
  /** Each person's ratings; the rating_* fields above are their averages. */
  scores: Score[]
}

export interface RestaurantSummary extends Restaurant {
  cover_key: string | null
  last_visit: string | null
  visit_count: number
}

export interface RestaurantDetail extends Restaurant {
  cover_key: string | null
  /** Picture taken from the website (og:image), if any. */
  site_photo: Photo | null
  visits: Visit[]
}

/** What POST /api/admin/site-preview found on a website. */
export interface SitePreview {
  title: string
  /** Pending picture, claimed by saving the restaurant with site_image_key. */
  image_key?: string
  width?: number
  height?: number
  note?: string
}

export interface RestaurantInput {
  name: string
  cuisine: string
  address: string
  city: string
  website: string
  price_level: number | null
  michelin_stars: number | null
  latitude: number | null
  longitude: number | null
  review: string
  scores: ScoreInput[]
  /** Creation only: the visit the restaurant starts with. */
  first_visit?: VisitInput
  /** A fetched website picture to keep; '' removes it, absent leaves it unchanged. */
  site_image_key?: string
}

export interface User {
  id: number
  email: string
  name: string
  created_at: string
}

export type SortMode = 'rank' | 'rating' | 'recent' | 'name'

export const RATING_CRITERIA = [
  { key: 'rating_food', label: 'Food' },
  { key: 'rating_service', label: 'Service' },
  { key: 'rating_ambiance', label: 'Ambiance' },
  { key: 'rating_value', label: 'Value' },
  /** Higher is quieter: 5 means you can talk easily. */
  { key: 'rating_quiet', label: 'Quiet' }
] as const

/** The public About page: Markdown already rendered (and sanitised) by the server. */
export interface AboutPage {
  html: string
  photo: string | null
}

export interface AdminAbout {
  markdown: string
  is_default: boolean
  default_markdown: string
  photo: string | null
}
