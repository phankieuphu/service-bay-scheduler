import { createContext, useContext } from 'react'
import type { RegisterPayload, User } from '../api/identity'

export type AuthState =
  // Restoring a stored session on page load.
  | { status: 'loading' }
  // A stored session exists but couldn't be checked (identity-service
  // unreachable). The refresh token is kept so retry() can still use it.
  | { status: 'unreachable'; message: string }
  | { status: 'signed-out'; notice?: string }
  | { status: 'signed-in'; user: User }

export interface AuthContextValue {
  state: AuthState
  signIn: (email: string, password: string) => Promise<void>
  // register creates the account and then signs straight in with it.
  register: (payload: RegisterPayload) => Promise<void>
  signOut: () => Promise<void>
  retry: () => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside <AuthProvider>')
  return value
}
