import { useState } from 'react'
import {
  formatMoney,
  newIdempotencyKey,
  payBill,
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

function paymentLine(p: Payment): string {
  const card = p.card_brand && p.card_last4 ? ` · ${p.card_brand} •••• ${p.card_last4}` : ''
  return `${p.type === 'DEPOSIT' ? 'Deposit' : 'Payment'} ${formatMoney(p.amount_minor, p.currency)}${card}`
}

interface Props {
  bill: Bill
  // Only the bill's customer can pay; staff see it read-only.
  canPay: boolean
  onChanged: () => void
}

export function BillView({ bill, canPay, onChanged }: Props) {
  const [started, setStarted] = useState<StartedPayment | null>(null)
  // One key per "Pay" action, reused if the click is retried after an error.
  const [payKey, setPayKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [starting, setStarting] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  const money = (minor: number) => formatMoney(minor, bill.currency)
  const payable = canPay && (bill.status === 'PENDING' || bill.status === 'PARTIALLY_PAID') && (bill.balance_due_minor ?? 0) > 0

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
    <div className="stack">
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
        <ul className="item-list">
          {bill.payments.map((p) => (
            <li key={p.id} className="item">
              <span className="small">{paymentLine(p)}</span>
              <span className="muted small">
                {p.status === 'SUCCEEDED'
                  ? 'paid'
                  : p.status === 'REQUIRES_PAYMENT' && p.failure_message
                    ? p.failure_message
                    : p.status.toLowerCase().replace('_', ' ')}
              </span>
            </li>
          ))}
        </ul>
      )}

      <dl className="details">
        <dt>Paid</dt>
        <dd>{money(bill.amount_paid_minor)}</dd>
        {bill.balance_due_minor !== null && (
          <>
            <dt>Balance due</dt>
            <dd className="strong">{money(bill.balance_due_minor)}</dd>
          </>
        )}
        {bill.refund_due_minor > 0 && (
          <>
            <dt>Refund due</dt>
            <dd>{money(bill.refund_due_minor)}</dd>
          </>
        )}
      </dl>

      {result && <p className="message success flush">{result}</p>}
      {error && <p className="message error flush">{error}</p>}

      {payable && !started && (
        <div className="actions">
          <button type="button" disabled={starting} onClick={startPayment}>
            {starting ? 'Preparing…' : `Pay ${money(bill.balance_due_minor ?? 0)}`}
          </button>
        </div>
      )}
      {started && (
        <PaymentForm started={started} label="Pay the balance" onSettled={settled} onCancel={() => setStarted(null)} />
      )}
    </div>
  )
}
