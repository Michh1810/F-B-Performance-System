import { apiRequest } from "./client"
import type { RestaurantProfile } from "./types"

// GET /api/restaurant-profile — internal/handlers/restaurantprofile.go
export function getRestaurantProfile(signal?: AbortSignal): Promise<RestaurantProfile> {
  return apiRequest<RestaurantProfile>("/api/restaurant-profile", { signal })
}

// PUT /api/restaurant-profile — internal/handlers/restaurantprofile.go
export function updateRestaurantProfile(description: string): Promise<RestaurantProfile> {
  return apiRequest<RestaurantProfile>("/api/restaurant-profile", {
    method: "PUT",
    body: { description },
  })
}
