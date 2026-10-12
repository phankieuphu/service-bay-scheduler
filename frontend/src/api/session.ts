import { ApiError, request } from './http'
import { refresh, type TokenResponse } from './identity'

// The refresh token lives in an HttpOnly cookie that only the browser can
// see, so this module can't tell whether one exists. SIGNED_IN_KEY is a
// non-secret flag that remembers it probably does, so a reload knows to
// try a refresh, and so other tabs can follow a sign-out. The access token
// is short-lived (15 min) and only ever held in memory.
const SIGNED_IN_KEY = 'service-bay.signed-in'

// The refresh token from before it moved to a cookie. It's removed on load
// and never read: it would still work until it expires, so leaving it in
// localStorage would leave it open to XSS.
const LEGACY_REFRESH_TOKEN_KEY = 'service-bay.refresh-token'

let accessToken: string | null = null
let inFlightRefresh: Promise<string> | null = null
const endListeners = new Set<() => void>()

function readSignedIn(): boolean {
  try {
    return localStorage.getItem(SIGNED_IN_KEY) !== null
  } catch {
    return false
  }
}

function writeSignedIn(signedIn: boolean) {
  try {
    if (signedIn) localStorage.setItem(SIGNED_IN_KEY, '1')
    else localStorage.removeItem(SIGNED_IN_KEY)
  } catch {
    // Storage blocked (private mode etc.): the session still works until
    // the tab is closed or reloaded.
  }
}

try {
  localStorage.removeItem(LEGACY_REFRESH_TOKEN_KEY)
} catch {
  // storage blocked; nothing to clean up
}

export function hasSession(): boolean {
  return accessToken !== null || readSignedIn()
}

export function startSession(tokens: TokenResponse) {
  accessToken = tokens.access_token
  writeSignedIn(true)
}

export function clearSession() {
  accessToken = null
  writeSignedIn(false)
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

// Another tab signing out removes the flag; follow it.
window.addEventListener('storage', (event) => {
  if (event.key === SIGNED_IN_KEY && event.newValue === null && accessToken !== null) {
    endSession()
  }
})

// inTabLock runs fn holding a lock shared by every tab of this origin, so
// only one tab refreshes at a time. Where the Web Locks API is missing it
// just runs fn.
function inTabLock<T>(fn: () => Promise<T>): Promise<T> {
  if (!navigator.locks) return fn()
  return navigator.locks.request('service-bay.refresh', fn)
}

// refreshAccessToken trades the refresh cookie for a new access token (and
// a rotated cookie). The API rotates refresh tokens — each one works exactly
// once — so concurrent callers in this tab (parallel 401s, StrictMode's
// double-run effects) share one request, and tabs take turns through a
// lock: a tab that waited sends the cookie the previous tab just got back.
export function refreshAccessToken(): Promise<string> {
  inFlightRefresh ??= (async () => {
    try {
      const tokens = await inTabLock(refresh)
      startSession(tokens)
      return tokens.access_token
    } catch (err) {
      // Only a definite rejection ends the session. A network error or 5xx
      // is transient: the cookie is still there for a retry.
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
