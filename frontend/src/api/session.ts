import { ApiError, request } from './http'
import { refresh, type TokenResponse } from './identity'

// The refresh token outlives a page reload, so it's kept in localStorage.
// The access token is short-lived (15 min) and only ever held in memory: a
// reload just trades the refresh token for a new one.
const REFRESH_TOKEN_KEY = 'service-bay.refresh-token'

let accessToken: string | null = null
let inFlightRefresh: Promise<string> | null = null
const endListeners = new Set<() => void>()

function readRefreshToken(): string | null {
  try {
    return localStorage.getItem(REFRESH_TOKEN_KEY)
  } catch {
    return null
  }
}

function writeRefreshToken(token: string | null) {
  try {
    if (token) localStorage.setItem(REFRESH_TOKEN_KEY, token)
    else localStorage.removeItem(REFRESH_TOKEN_KEY)
  } catch {
    // Storage blocked (private mode etc.): the session still works until
    // the tab is closed or reloaded.
  }
}

export function hasSession(): boolean {
  return accessToken !== null || readRefreshToken() !== null
}

export function getRefreshToken(): string | null {
  return readRefreshToken()
}

export function startSession(tokens: TokenResponse) {
  accessToken = tokens.access_token
  writeRefreshToken(tokens.refresh_token)
}

export function clearSession() {
  accessToken = null
  writeRefreshToken(null)
}

// onSessionEnded fires when the session is lost involuntarily: the refresh
// token was rejected (expired, revoked, or used by another tab), or another
// tab signed out. Returns an unsubscribe function.
export function onSessionEnded(listener: () => void): () => void {
  endListeners.add(listener)
  return () => endListeners.delete(listener)
}

function endSession() {
  clearSession()
  endListeners.forEach((listener) => listener())
}

// Another tab signing out removes the shared refresh token; follow it.
window.addEventListener('storage', (event) => {
  if (event.key === REFRESH_TOKEN_KEY && event.newValue === null && accessToken !== null) {
    endSession()
  }
})

// refreshAccessToken trades the stored refresh token for a new pair. The API
// rotates refresh tokens — each one works exactly once — so concurrent
// callers (parallel 401s, StrictMode's double-run effects) must share one
// request; a second request with the same token would be rejected and sign
// the user out.
export function refreshAccessToken(): Promise<string> {
  inFlightRefresh ??= (async () => {
    const refreshToken = readRefreshToken()
    if (!refreshToken) {
      endSession()
      throw new ApiError(401, 'Not signed in.')
    }
    try {
      const tokens = await refresh(refreshToken)
      startSession(tokens)
      return tokens.access_token
    } catch (err) {
      // Only a definite rejection ends the session. A network error or 5xx
      // is transient: keep the refresh token so a retry can still use it.
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        endSession()
      }
      throw err
    } finally {
      inFlightRefresh = null
    }
  })()
  return inFlightRefresh
}

// withAccessToken calls fn with a valid access token, refreshing first if
// there isn't one, and refreshing once more and retrying if fn gets a 401
// (the token expired between calls).
export async function withAccessToken<T>(fn: (token: string) => Promise<T>): Promise<T> {
  const token = accessToken ?? (await refreshAccessToken())
  try {
    return await fn(token)
  } catch (err) {
    if (!(err instanceof ApiError && err.status === 401)) throw err
    return fn(await refreshAccessToken())
  }
}

// authorizedRequest is request() with the signed-in user's bearer token.
export function authorizedRequest<T>(url: string, init: RequestInit = {}): Promise<T> {
  return withAccessToken((token) => {
    const headers = new Headers(init.headers)
    headers.set('Authorization', `Bearer ${token}`)
    return request<T>(url, { ...init, headers })
  })
}
