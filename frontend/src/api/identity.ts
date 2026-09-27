import { jsonBody, request } from './http'

export { ApiError } from './http'

const BASE_URL =
  import.meta.env.VITE_IDENTITY_API_URL ?? 'http://localhost:8083/api/v1'

export type Role = 'CUSTOMER' | 'TECHNICIAN' | 'MANAGER' | 'ADMIN'

// Only these can be chosen at sign-up; the API rejects MANAGER/ADMIN with 403.
export type SelfRegisterRole = Extract<Role, 'CUSTOMER' | 'TECHNICIAN'>

export type UserStatus = 'ACTIVE' | 'INACTIVE' | 'LOCKED'

export interface User {
  id: string
  email: string
  role: Role
  status: UserStatus
  created_at: string
}

export interface TokenResponse {
  access_token: string
  token_type: 'Bearer'
  expires_in: number
  refresh_token: string
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

export function login(email: string, password: string): Promise<TokenResponse> {
  return request(`${BASE_URL}/auth/login`, jsonBody('POST', { email, password }))
}

export function refresh(refreshToken: string): Promise<TokenResponse> {
  return request(`${BASE_URL}/auth/refresh`, jsonBody('POST', { refresh_token: refreshToken }))
}

export function logout(refreshToken: string): Promise<void> {
  return request(`${BASE_URL}/auth/logout`, jsonBody('POST', { refresh_token: refreshToken }))
}

export function getMe(accessToken: string): Promise<User> {
  return request(`${BASE_URL}/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
}
