import { useEffect, useState, type FormEvent } from 'react'
import {
  ApiError,
  getCustomer,
  listCustomers,
  type Customer,
  type CustomerPage,
} from '../api/customer'
import { CustomerDetail } from './CustomerDetail'

const PAGE_SIZE = 10

export function CustomerList() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [nextCursor, setNextCursor] = useState<number | null>(null)
  const [selected, setSelected] = useState<Customer | null>(null)
  const [lookupId, setLookupId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  function applyPage(page: CustomerPage, cursor: number) {
    setCustomers((prev) => (cursor === 0 ? page.customers : [...prev, ...page.customers]))
    setNextCursor(page.has_more && page.next_cursor ? page.next_cursor : null)
  }

  function applyError(err: unknown) {
    setError(err instanceof Error ? err.message : 'Failed to load customers.')
  }

  async function loadPage(cursor: number) {
    setError(null)
    setLoading(true)
    try {
      applyPage(await listCustomers(cursor, PAGE_SIZE), cursor)
    } catch (err) {
      applyError(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    listCustomers(0, PAGE_SIZE)
      .then((page) => applyPage(page, 0))
      .catch(applyError)
      .finally(() => setLoading(false))
  }, [])

  async function handleLookup(event: FormEvent) {
    event.preventDefault()
    setError(null)
    setNotice(null)

    const id = Number(lookupId)
    if (!Number.isInteger(id) || id <= 0) {
      setError('Enter a valid customer ID.')
      return
    }

    try {
      setSelected(await getCustomer(id))
    } catch (err) {
      setSelected(null)
      if (err instanceof ApiError && err.status === 404) {
        setError(`No customer found with ID ${id}.`)
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong.')
      }
    }
  }

  function handleUpdated(updated: Customer) {
    setSelected(updated)
    setCustomers((prev) => prev.map((c) => (c.id === updated.id ? updated : c)))
  }

  function handleDeleted(customerId: number) {
    setSelected(null)
    setCustomers((prev) => prev.filter((c) => c.id !== customerId))
    setNotice(`Customer ${customerId} deleted.`)
  }

  return (
    <div className="panel">
      <div className="panel-header">
        <h2>Customers</h2>
        <form className="form-row" onSubmit={handleLookup}>
          <label htmlFor="customer-lookup-id">Customer ID</label>
          <input
            id="customer-lookup-id"
            type="number"
            min={1}
            value={lookupId}
            onChange={(e) => setLookupId(e.target.value)}
            placeholder="Look up by ID"
          />
          <button type="submit">Look up</button>
        </form>
      </div>

      {error && <p className="message error">{error}</p>}
      {notice && <p className="message success">{notice}</p>}

      {selected && (
        <CustomerDetail
          key={selected.id}
          customer={selected}
          onUpdated={handleUpdated}
          onDeleted={handleDeleted}
        />
      )}

      <table className="list">
        <thead>
          <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Email</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {customers.map((customer) => (
            <tr
              key={customer.id}
              className={selected?.id === customer.id ? 'selected' : ''}
              onClick={() => {
                setError(null)
                setNotice(null)
                setSelected(customer)
              }}
            >
              <td>{customer.id}</td>
              <td>{customer.name}</td>
              <td>{customer.email}</td>
              <td>
                <span className={`badge badge-${customer.status.toLowerCase()}`}>
                  {customer.status}
                </span>
              </td>
            </tr>
          ))}
          {!loading && customers.length === 0 && (
            <tr>
              <td colSpan={4} className="muted empty">
                No customers yet.
              </td>
            </tr>
          )}
        </tbody>
      </table>

      <div className="actions list-footer">
        {nextCursor !== null && (
          <button
            type="button"
            className="secondary"
            disabled={loading}
            onClick={() => loadPage(nextCursor)}
          >
            {loading ? 'Loading…' : 'Load more'}
          </button>
        )}
        <button
          type="button"
          className="secondary"
          disabled={loading}
          onClick={() => loadPage(0)}
        >
          Refresh
        </button>
      </div>
    </div>
  )
}
