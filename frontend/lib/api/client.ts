// Shared fetch wrapper for the Go API (cmd/api). Every module under
// lib/api/ routes its requests through `apiRequest` instead of calling
// `fetch` directly, so base-URL resolution and error handling stay in one
// place.

export function getApiBaseUrl(): string {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"
}

export class ApiError extends Error {
  status: number
  /** True when the backend has no route for this request yet (404/405) — see PLACEHOLDER_ENDPOINT in ideas.ts. */
  notImplemented: boolean

  constructor(message: string, status: number) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.notImplemented = status === 404 || status === 405
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE"
  body?: unknown
  signal?: AbortSignal
}

export async function apiRequest<T>(
  path: string,
  { method = "GET", body, signal }: RequestOptions = {}
): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    cache: "no-store",
    signal,
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response)
    throw new ApiError(message, response.status)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

async function extractErrorMessage(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as { error?: string }
    if (data?.error) {
      return data.error
    }
  } catch {
    // response body wasn't JSON — fall through to the status text below
  }
  return `${response.status} ${response.statusText}`
}
