import { apiRequest } from "./client"
import type { ActiveMenuItem } from "./types"

// GET /api/menu-items/active — internal/handlers/menuitems.go, wrapping the
// same MenuItemStore.ListActive the Menu Idea Agent scans every run.
export function listActiveMenuItems(signal?: AbortSignal): Promise<ActiveMenuItem[]> {
  return apiRequest<ActiveMenuItem[]>("/api/menu-items/active", { signal })
}
