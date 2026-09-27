import { useState, type FormEvent } from 'react'
import {
  PASSWORD_MAX_BYTES,
  PASSWORD_MIN_LENGTH,
  type SelfRegisterRole,
} from '../api/identity'
import { useAuth } from '../auth/auth-context'
import { describeAuthError } from '../auth/errors'

interface FormState {
  email: string
  password: string
  confirmPassword: string
  role: SelfRegisterRole
}

const initialState: FormState = { email: '', password: '', confirmPassword: '', role: 'CUSTOMER' }

const roles: { id: SelfRegisterRole; label: string }[] = [
  { id: 'CUSTOMER', label: 'Customer' },
  { id: 'TECHNICIAN', label: 'Technician' },
]

export function SignUpForm() {
  const { register } = useAuth()
  const [form, setForm] = useState<FormState>(initialState)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function updateField<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)

    if (form.password.length < PASSWORD_MIN_LENGTH) {
      setError(`Password must be at least ${PASSWORD_MIN_LENGTH} characters.`)
      return
    }
    if (new TextEncoder().encode(form.password).length > PASSWORD_MAX_BYTES) {
      setError(`Password is too long (max ${PASSWORD_MAX_BYTES} bytes).`)
      return
    }
    if (form.password !== form.confirmPassword) {
      setError('Passwords don’t match.')
      return
    }

    setLoading(true)
    try {
      await register({ email: form.email.trim(), password: form.password, role: form.role })
      // Signed in straight away; AuthProvider switches the app over.
    } catch (err) {
      setError(describeAuthError(err))
      setLoading(false)
    }
  }

  return (
    <form className="form-grid" onSubmit={handleSubmit}>
      <label htmlFor="sign-up-email">Email</label>
      <input
        id="sign-up-email"
        type="email"
        autoComplete="username"
        maxLength={255}
        value={form.email}
        onChange={(e) => updateField('email', e.target.value)}
        required
      />

      <label htmlFor="sign-up-password">Password</label>
      <input
        id="sign-up-password"
        type="password"
        autoComplete="new-password"
        minLength={PASSWORD_MIN_LENGTH}
        value={form.password}
        onChange={(e) => updateField('password', e.target.value)}
        aria-describedby="sign-up-password-hint"
        required
      />
      <span id="sign-up-password-hint" className="hint span-2 full">
        At least {PASSWORD_MIN_LENGTH} characters.
      </span>

      <label htmlFor="sign-up-confirm">Confirm password</label>
      <input
        id="sign-up-confirm"
        type="password"
        autoComplete="new-password"
        value={form.confirmPassword}
        onChange={(e) => updateField('confirmPassword', e.target.value)}
        required
      />

      <span className="label" id="sign-up-role-label">
        I am a
      </span>
      <div className="segmented" role="radiogroup" aria-labelledby="sign-up-role-label">
        {roles.map((role) => (
          <label key={role.id} className={form.role === role.id ? 'active' : ''}>
            <input
              type="radio"
              name="role"
              value={role.id}
              checked={form.role === role.id}
              onChange={() => updateField('role', role.id)}
            />
            {role.label}
          </label>
        ))}
      </div>

      <button type="submit" disabled={loading} className="span-2">
        {loading ? 'Creating account…' : 'Create account'}
      </button>

      {error && <p className="message error span-2 full">{error}</p>}
    </form>
  )
}
