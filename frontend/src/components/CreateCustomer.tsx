import { useState, type FormEvent } from 'react'
import { ApiError, createCustomer, inputToBirthDay } from '../api/customer'

interface FormState {
  name: string
  email: string
  phone: string
  birthDay: string
}

const initialState: FormState = { name: '', email: '', phone: '', birthDay: '' }

export function CreateCustomer() {
  const [form, setForm] = useState<FormState>(initialState)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function updateField<K extends keyof FormState>(key: K, value: string) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    if (form.birthDay > new Date().toISOString().slice(0, 10)) {
      setError('Birthday must be in the past.')
      return
    }

    setLoading(true)
    try {
      const created = await createCustomer({
        name: form.name.trim(),
        email: form.email.trim(),
        phone: form.phone.trim() || undefined,
        birth_day: inputToBirthDay(form.birthDay),
      })
      setSuccess(`Customer ${created.name} created with ID ${created.id}.`)
      setForm(initialState)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError('A customer with this email already exists.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="panel">
      <h2>Create Customer</h2>
      <form className="form-grid" onSubmit={handleSubmit}>
        <label htmlFor="create-name">Name</label>
        <input
          id="create-name"
          value={form.name}
          onChange={(e) => updateField('name', e.target.value)}
          required
        />

        <label htmlFor="create-email">Email</label>
        <input
          id="create-email"
          type="email"
          value={form.email}
          onChange={(e) => updateField('email', e.target.value)}
          required
        />

        <label htmlFor="create-phone">Phone (optional)</label>
        <input
          id="create-phone"
          type="tel"
          value={form.phone}
          onChange={(e) => updateField('phone', e.target.value)}
        />

        <label htmlFor="create-birth-day">Birthday</label>
        <input
          id="create-birth-day"
          type="date"
          value={form.birthDay}
          onChange={(e) => updateField('birthDay', e.target.value)}
          required
        />

        <button type="submit" disabled={loading} className="span-2">
          {loading ? 'Creating…' : 'Create'}
        </button>
      </form>

      {error && <p className="message error">{error}</p>}
      {success && <p className="message success">{success}</p>}
    </div>
  )
}
