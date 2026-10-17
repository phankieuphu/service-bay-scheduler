import { useState } from 'react'
import {
  formatMoney,
  newIdempotencyKey,
  payBill,
  syncPayment,
  type Bill,
  type BillStatus,
  type Payment,
  type StartedPayment,
} from '../api/billing'
import { PaymentForm } from './PaymentForm'

const billBadge: Record<BillStatus, string> = {
  OPEN: 'badge-warn',
  PENDING: 'badge-warn',
  PARTIALLY_PAID: 'badge-warn',
  PAID: 'badge-active',
  VOID: 'badge-scrapped',
}

const billStatusLabel: Record<BillStatus, string> = {
  OPEN: 'Not priced yet',
  PENDING: 'Due',
  PARTIALLY_PAID: 'Partly paid',
  PAID: 'Paid',
  VOID: 'Void',
}

function paymentBadge(p: Payment): { className: string; label: string } {
  switch (p.status) {
    case 'SUCCEEDED':
      return { className: 'badge-active', label: 'Paid' }
    case 'REQUIRES_PAYMENT':
      // A declined attempt stays REQUIRES_PAYMENT so it can be retried.
      return p.failure_message
        ? { className: 'badge-scrapped', label: 'Declined' }
        : { className: 'badge-warn', label: 'Awaiting payment' }
    case 'PROCESSING':
      return { className: 'badge-warn', label: 'Processing' }
    case 'FAILED':
      return { className: 'badge-scrapped', label: 'Failed' }
    case 'CANCELED':
      return { className: 'badge-inactive', label: 'Canceled' }
    case 'REFUNDED':
      return { className: 'badge-inactive', label: 'Refunded' }
    case 'PARTIALLY_REFUNDED':
      return { className: 'badge-inactive', label: 'Partly refunded' }
  }
}

// Only these can still change on Stripe's side, so only these are worth syncing.
function isUnsettled(p: Payment): boolean {
  return p.status === 'REQUIRES_PAYMENT' || p.status === 'PROCESSING'
}

function paymentDetail(p: Payment): string | null {
  const parts: string[] = []
  if (p.card_brand && p.card_last4) parts.push(`${p.card_brand.toUpperCase()} •••• ${p.card_last4}`)
  if (p.paid_at) parts.push(new Date(p.paid_at).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }))
  if (p.status === 'REQUIRES_PAYMENT' && p.failure_message) parts.push(p.failure_message)
  return parts.length > 0 ? parts.join(' · ') : null
}

interface Props {
  bill: Bill
  // Only the bill's customer can pay; staff see it read-only.
  canPay: boolean
  // Staff can re-read an unsettled payment from Stripe when its webhook was missed.
  canSync?: boolean
  onChanged: () => void
}

export function BillView({ bill, canPay, canSync = false, onChanged }: Props) {
  const [started, setStarted] = useState<StartedPayment | null>(null)
  // One key per "Pay" action, reused if the click is retried after an error.
  const [payKey, setPayKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [starting, setStarting] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const money = (minor: number) => formatMoney(minor, bill.currency)
  const balanceDue = bill.balance_due_minor ?? 0
  const payable = canPay && (bill.status === 'PENDING' || bill.status === 'PARTIALLY_PAID') && balanceDue > 0

  async function startPayment() {
    const key = payKey ?? newIdempotencyKey()
    setPayKey(key)
    setStarting(true)
    setError(null)
    setResult(null)
    try {
      setStarted(await payBill(bill.id, key))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Couldn’t start the payment.')
    } finally {
      setStarting(false)
    }
  }

  function settled(payment: Payment) {
    setStarted(null)
    setPayKey(null)
    if (payment.status === 'SUCCEEDED') {
      setResult(`Paid ${formatMoney(payment.amount_minor, payment.currency)}. Thank you!`)
    } else if (payment.status === 'PROCESSING' || payment.status === 'REQUIRES_PAYMENT') {
      setResult('Your payment is still being confirmed. The bill updates once it goes through.')
    } else {
      setError(payment.failure_message ?? `The payment ended as ${payment.status.toLowerCase()}.`)
    }
    onChanged()
  }

  return (
    <div className="stack bill">
      <div className="subsection-header">
        <h3>Bill #{bill.id}</h3>
        <span className={`badge ${billBadge[bill.status]}`}>{billStatusLabel[bill.status]}</span>
      </div>

      {bill.status === 'OPEN' ? (
        <p className="muted small flush">
          The bill is priced when the service is completed. Anything paid now counts toward it.
        </p>
      ) : (
        <table className="bill-lines">
          <tbody>
            {bill.lines.map((line) => (
              <tr key={line.id}>
                <td>
                  {line.name}
                  {line.quantity > 1 && <span className="muted"> × {line.quantity}</span>}
                </td>
                <td className="amount">{money(line.line_total_minor)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <th scope="row">Total</th>
              <td className="amount strong">{money(bill.total_minor ?? 0)}</td>
            </tr>
          </tfoot>
        </table>
      )}

      {bill.payments.length > 0 && (
        <div className="stack tight">
          <span className="label">Payments</span>
          <ul className="item-list">
            {bill.payments.map((p) => (
              <PaymentRow key={p.id} payment={p} canSync={canSync && isUnsettled(p)} onSynced={onChanged} />
            ))}
          </ul>
        </div>
      )}

      <div className="bill-summary">
        <dl className="bill-totals">
          <div>
            <dt>Paid</dt>
            <dd>{money(bill.amount_paid_minor)}</dd>
          </div>
          {bill.refund_due_minor > 0 && (
            <div>
              <dt>Refund due</dt>
              <dd>{money(bill.refund_due_minor)}</dd>
            </div>
          )}
          {bill.balance_due_minor !== null && (
            <div className="balance">
              <dt>Balance due</dt>
              <dd>{money(bill.balance_due_minor)}</dd>
            </div>
          )}
        </dl>
        {payable && !started && (
          <button type="button" className="primary" disabled={starting} onClick={startPayment}>
            {starting ? 'Preparing…' : `Pay ${money(balanceDue)}`}
          </button>
        )}
      </div>

      {result && <p className="message success flush">{result}</p>}
      {error && <p className="message error flush">{error}</p>}

      {started && (
        <PaymentForm started={started} label="Pay the balance" onSettled={settled} onCancel={() => setStarted(null)} />
      )}
    </div>
  )
}

function PaymentRow({ payment, canSync, onSynced }: { payment: Payment; canSync: boolean; onSynced: () => void }) {
  const [syncing, setSyncing] = useState(false)
  const [note, setNote] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const badge = paymentBadge(payment)
  const detail = paymentDetail(payment)

  async function sync() {
    setSyncing(true)
    setError(null)
    setNote(null)
    try {
      const updated = await syncPayment(payment.id)
      if (updated.status === payment.status) {
        setNote(`Stripe still reports it as ${paymentBadge(updated).label.toLowerCase()}.`)
      } else {
        onSynced()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Couldn’t sync with Stripe.')
    } finally {
      setSyncing(false)
    }
  }

  return (
    <li className="item payment-row">
      <div className="item-main">
        <span>
          {payment.type === 'DEPOSIT' ? 'Deposit' : 'Payment'}{' '}
          <span className="strong amount">{formatMoney(payment.amount_minor, payment.currency)}</span>
        </span>
        {detail && <span className="muted small">{detail}</span>}
        {note && <span className="muted small">{note}</span>}
        {error && <span className="field-error">{error}</span>}
      </div>
      <div className="item-meta">
        <span className={`badge ${badge.className}`}>{badge.label}</span>
        {canSync && (
          <button
            type="button"
            className="secondary"
            disabled={syncing}
            onClick={sync}
            title="Re-read this payment from Stripe, in case its webhook was missed"
          >
            {syncing ? 'Syncing…' : 'Sync with Stripe'}
          </button>
        )}
      </div>
    </li>
  )
}
