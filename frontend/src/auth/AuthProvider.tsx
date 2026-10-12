import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import * as identity from '../api/identity'
import {
  clearSession,
  hasSession,
  onSessionEnded,
  startSession,
  withAccessToken,
} from '../api/session'
import { AuthContext, type AuthContextValue, type AuthState } from './auth-context'

const SESSION_EXPIRED = 'Your session has expired. Please sign in again.'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() =>
    hasSession() ? { status: 'loading' } : { status: 'signed-out' },
  )
  const [attempt, setAttempt] = useState(0)

  // Restore a stored session: refresh the access token, then load the user.
  useEffect(() => {
    if (!hasSession()) return
    let cancelled = false
    withAccessToken(identity.getMe)
      .then((user) => {
        if (!cancelled) setState({ status: 'signed-in', user })
      })
      .catch((err) => {
        if (cancelled) return
        if (!hasSession()) {
          // The refresh cookie was rejected (or there wasn't one).
          setState({ status: 'signed-out', notice: SESSION_EXPIRED })
        } else {
          setState({
            status: 'unreachable',
            message: err instanceof Error ? err.message : 'Something went wrong.',
          })
        }
      })
    return () => {
      cancelled = true
    }
  }, [attempt])

  useEffect(
    () => onSessionEnded(() => setState({ status: 'signed-out', notice: SESSION_EXPIRED })),
    [],
  )

  const signIn = useCallback(async (email: string, password: string) => {
    startSession(await identity.login(email, password))
    try {
      const user = await withAccessToken(identity.getMe)
      setState({ status: 'signed-in', user })
    } catch (err) {
      clearSession()
      throw err
    }
  }, [])

  const register = useCallback(
    async (payload: identity.RegisterPayload) => {
      await identity.register(payload)
      await signIn(payload.email, payload.password)
    },
    [signIn],
  )

  const signOut = useCallback(async () => {
    clearSession()
    setState({ status: 'signed-out' })
    // Revokes the refresh token and clears its cookie. If this fails the
    // token still expires on its own, so don't surface it.
    await identity.logout().catch(() => undefined)
  }, [])

  const retry = useCallback(() => {
    setState({ status: 'loading' })
    setAttempt((n) => n + 1)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({ state, signIn, register, signOut, retry }),
    [state, signIn, register, signOut, retry],
  )

  return <AuthContext value={value}>{children}</AuthContext>
}
