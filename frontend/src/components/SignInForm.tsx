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
    <form className="stack" onSubmit={handleSubmit}>
      <label className="field" htmlFor="sign-in-email">
        Email
        <input
          id="sign-in-email"
          type="email"
          autoComplete="username"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
      </label>

      <label className="field" htmlFor="sign-in-password">
        Password
        <input
          id="sign-in-password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
      </label>

      {error && <p className="message error flush">{error}</p>}

      <button type="submit" disabled={loading} className="align-start">
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  )
}
