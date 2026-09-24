import { useState, type FormEvent } from 'react'
import {
  ApiError,
  birthDayToInput,
  deleteCustomer,
  getCustomer,
  inputToBirthDay,
  updateCustomer,
  type Customer,
} from '../api/customer'

interface Props {
  customer: Customer
  onUpdated: (customer: Customer) => void
  onDeleted: (customerId: number) => void
}

type Mode = 'view' | 'edit' | 'confirm-delete'

export function CustomerDetail({ customer, onUpdated, onDeleted }: Props) {
  const [mode, setMode] = useState<Mode>('view')
  const [name, setName] = useState(customer.name)
  const [birthDay, setBirthDay] = useState(birthDayToInput(customer.birth_day))
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  function startEdit() {
    setName(customer.name)
    setBirthDay(birthDayToInput(customer.birth_day))
    setError(null)
    setSuccess(null)
    setMode('edit')
  }

  // Reload the latest version after a 409 so the user can retry their edit
  // against the current data instead of looping on stale updated_at.
  async function reload() {
    const latest = await getCustomer(customer.id)
    onUpdated(latest)
    setName(latest.name)
    setBirthDay(birthDayToInput(latest.birth_day))
  }

  async function handleUpdate(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setSuccess(null)

    const trimmedName = name.trim()
    const nameChanged = trimmedName !== customer.name
    const birthDayChanged = birthDay !== birthDayToInput(customer.birth_day)

    if (!trimmedName) {
      setError('Name cannot be empty.')
      return
    }
    if (!nameChanged && !birthDayChanged) {
      setError('Nothing to update.')
      return
    }
    if (birthDay && birthDay > new Date().toISOString().slice(0, 10)) {
      setError('Birthday must be in the past.')
      return
    }

    setLoading(true)
    try {
      await updateCustomer(customer.id, {
        name: nameChanged ? trimmedName : undefined,
        birth_day: birthDayChanged && birthDay ? inputToBirthDay(birthDay) : undefined,
        updated_at: customer.updated_at,
      })
      onUpdated(await getCustomer(customer.id))
      setSuccess('Customer updated.')
      setMode('view')
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError(
          'This customer was changed by someone else since you loaded it. The latest version has been loaded — review and save again.',
        )
        await reload().catch(() => undefined)
      } else if (err instanceof ApiError && err.status === 404) {
        setError('This customer no longer exists.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete() {
    setError(null)
    setSuccess(null)
    setLoading(true)
    try {
      await deleteCustomer(customer.id)
      onDeleted(customer.id)
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('This customer no longer exists.')
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
      setMode('view')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="detail-card">
      <div className="detail-header">
        <h3>
          {customer.name} <span className="muted">#{customer.id}</span>
        </h3>
        {mode === 'view' && (
          <div className="actions">
            <button type="button" className="secondary" onClick={startEdit}>
              Edit
            </button>
            <button
              type="button"
              className="danger"
              onClick={() => {
                setError(null)
                setSuccess(null)
                setMode('confirm-delete')
              }}
            >
              Delete
            </button>
          </div>
        )}
      </div>

      {mode === 'edit' ? (
        <form className="form-grid" onSubmit={handleUpdate}>
          <label htmlFor="edit-name">Name</label>
          <input
            id="edit-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />

          <label htmlFor="edit-birth-day">Birthday</label>
          <input
            id="edit-birth-day"
            type="date"
            value={birthDay}
            onChange={(e) => setBirthDay(e.target.value)}
          />

          <div className="span-2 actions">
            <button type="submit" disabled={loading}>
              {loading ? 'Saving…' : 'Save'}
            </button>
            <button
              type="button"
              className="secondary"
              disabled={loading}
              onClick={() => setMode('view')}
            >
              Cancel
            </button>
          </div>
        </form>
      ) : (
        <table className="details">
          <tbody>
            <tr>
              <th>Email</th>
              <td>{customer.email}</td>
            </tr>
            <tr>
              <th>Phone</th>
              <td>{customer.phone || '—'}</td>
            </tr>
            <tr>
              <th>Birthday</th>
              <td>{birthDayToInput(customer.birth_day)}</td>
            </tr>
            <tr>
              <th>Status</th>
              <td>
                <span className={`badge badge-${customer.status.toLowerCase()}`}>
                  {customer.status}
                </span>
              </td>
            </tr>
            <tr>
              <th>Created</th>
              <td>{new Date(customer.created_at).toLocaleString()}</td>
            </tr>
            <tr>
              <th>Updated</th>
              <td>{new Date(customer.updated_at).toLocaleString()}</td>
            </tr>
          </tbody>
        </table>
      )}

      {mode === 'confirm-delete' && (
        <div className="message error confirm">
          <span>
            Delete {customer.name}? They will no longer appear in listings or
            lookups.
          </span>
          <div className="actions">
            <button
              type="button"
              className="danger"
              disabled={loading}
              onClick={handleDelete}
            >
              {loading ? 'Deleting…' : 'Yes, delete'}
            </button>
            <button
              type="button"
              className="secondary"
              disabled={loading}
              onClick={() => setMode('view')}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {error && <p className="message error">{error}</p>}
      {success && <p className="message success">{success}</p>}
    </div>
  )
}
