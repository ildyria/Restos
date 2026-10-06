export type Meal = 'lunch' | 'dinner'
export type Course = 'starter' | 'main' | 'dessert' | 'drink' | 'other'

export interface Photo {
  id: number
  dish_id: number
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
  overall: number | null
  review: string
  rank_position: number | null
  cover_photo_id: number | null
  created_at: string
  updated_at: string
}

export interface RestaurantSummary extends Restaurant {
  cover_key: string | null
  last_visit: string | null
  visit_count: number
}

export interface RestaurantDetail extends Restaurant {
  cover_key: string | null
  visits: Visit[]
}

export interface RestaurantInput {
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
  review: string
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
  { key: 'rating_value', label: 'Value' }
] as const
