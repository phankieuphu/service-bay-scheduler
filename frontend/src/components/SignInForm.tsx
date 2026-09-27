import { useState, type FormEvent } from 'react'
import { useAuth } from '../auth/auth-context'
import { describeAuthError } from '../auth/errors'

export function SignInForm() {
  const { signIn } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await signIn(email.trim(), password)
      // On success AuthProvider switches the app over; this form unmounts.
    } catch (err) {
      setError(describeAuthError(err))
      setPassword('')
      setLoading(false)
    }
  }

  return (
    <form className="form-grid" onSubmit={handleSubmit}>
      <label htmlFor="sign-in-email">Email</label>
      <input
        id="sign-in-email"
        type="email"
        autoComplete="username"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        required
      />

      <label htmlFor="sign-in-password">Password</label>
      <input
        id="sign-in-password"
        type="password"
        autoComplete="current-password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        required
      />

      <button type="submit" disabled={loading} className="span-2">
        {loading ? 'Signing in…' : 'Sign in'}
      </button>

      {error && <p className="message error span-2 full">{error}</p>}
    </form>
  )
}
