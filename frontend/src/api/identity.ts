import { jsonBody, request } from './http'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_IDENTITY_API_URL ?? 'http://localhost:8083/api/v1'

export type Role = 'CUSTOMER' | 'TECHNICIAN' | 'MANAGER' | 'ADMIN'

// Only these can be chosen at sign-up; the API rejects MANAGER/ADMIN with 403.
// Staff roles (TECHNICIAN, MANAGER, ADMIN) are provisioned by an admin.
export type SelfRegisterRole = Extract<Role, 'CUSTOMER'>

export type UserStatus = 'ACTIVE' | 'INACTIVE' | 'LOCKED'

export interface User {
  id: string
  email: string
  role: Role
  status: UserStatus
  created_at: string
}

// The refresh token isn't here: the API sets it as an HttpOnly cookie
// scoped to /auth, so scripts (and any XSS) can never read it.
export interface TokenResponse {
  access_token: string
  token_type: 'Bearer'
  expires_in: number
}

export interface RegisterPayload {
  email: string
  password: string
  role?: SelfRegisterRole
}

// Mirrors the API's binding rules, so the form can explain them instead of
// surfacing gin's raw validation message. The 72 is bytes, not characters:
// bcrypt ignores anything past 72 bytes of input.
export const PASSWORD_MIN_LENGTH = 8
export const PASSWORD_MAX_BYTES = 72

export function register(payload: RegisterPayload): Promise<User> {
  return request(`${BASE_URL}/auth/register`, jsonBody('POST', payload))
}

// credentials: 'include' lets the browser store and send the refresh-token
// cookie on these cross-origin calls. The API requires X-Requested-With on
// the routes that read the cookie (a CSRF guard a cross-site form can't
// satisfy).
const withCookie: RequestInit = {
  credentials: 'include',
  headers: { 'X-Requested-With': 'fetch' },
}

export function login(email: string, password: string): Promise<TokenResponse> {
  return request(`${BASE_URL}/auth/login`, {
    ...jsonBody('POST', { email, password }),
    credentials: 'include',
  })
}

export function refresh(): Promise<TokenResponse> {
  return request(`${BASE_URL}/auth/refresh`, { ...withCookie, method: 'POST' })
}

export function logout(): Promise<void> {
  return request(`${BASE_URL}/auth/logout`, { ...withCookie, method: 'POST' })
}

export function getMe(accessToken: string): Promise<User> {
  return request(`${BASE_URL}/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
}
