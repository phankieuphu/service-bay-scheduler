import { useState } from 'react'
import { todayInput } from '../../api/vehicle'
import { HUBS, REPORT, formatVnd } from '../../demo/data'

function toCsv(): string {
  const rows = [
    ['section', 'name', 'value'],
    ...REPORT.kpis.map((k) => ['kpi', k.label, k.value]),
    ...REPORT.revenue.map((r) => ['revenue', r.name, String(r.amount)]),
    ...REPORT.technicians.map((t) => ['utilization', t.name, `${t.utilization}%`]),
    ...REPORT.materials.map((m) => ['material', m.name, m.used]),
  ]
  return rows.map((r) => r.map((c) => `"${c.replaceAll('"', '""')}"`).join(',')).join('\n')
}

function downloadCsv(day: string) {
  const url = URL.createObjectURL(new Blob([toCsv()], { type: 'text/csv' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `day-end-report-${day}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

export function ReportPage() {
  const [hub, setHub] = useState(String(HUBS[0].id))
  const [day, setDay] = useState(todayInput())
  const maxRevenue = Math.max(...REPORT.revenue.map((r) => r.amount))

  return (
    <div className="stack roomy demo">
      <div className="section-head wrap end">
        <div>
          <h2 className="demo-title">End-of-day report</h2>
          <p className="muted flush">Business day closes 23:59 Asia/Ho_Chi_Minh · sample data</p>
        </div>
        <div className="actions end">
          <label className="field">
            <span className="label">Hub</span>
            <select value={hub} onChange={(e) => setHub(e.target.value)}>
              {HUBS.map((h) => (
                <option key={h.id} value={h.id}>
                  {h.name}
                </option>
              ))}
              <option value="all">All hubs</option>
            </select>
          </label>
          <label className="field">
            <span className="label">Day</span>
            <input type="date" value={day} onChange={(e) => setDay(e.target.value)} />
          </label>
          <button type="button" className="secondary" onClick={() => downloadCsv(day)}>
            Export CSV
          </button>
        </div>
      </div>

      <section aria-label="Key figures" className="kpi-grid">
        {REPORT.kpis.map((k) => (
          <div key={k.label} className="card tight stack kpi">
            <span className="small muted">{k.label}</span>
            <strong className="kpi-value num">{k.value}</strong>
            <span className={`small tone-${k.tone}`}>{k.delta}</span>
          </div>
        ))}
      </section>

      <div className="two-col">
        <section className="card stack" aria-labelledby="revenue-heading">
          <h3 id="revenue-heading" className="demo-h3">
            Revenue by service type
          </h3>
          {REPORT.revenue.map((r) => (
            <div key={r.name} className="bar-row">
              <span>{r.name}</span>
              <span className="meter-track">
                <span className="meter-fill" style={{ width: `${(r.amount / maxRevenue) * 100}%` }} />
              </span>
              <span className="right num">{formatVnd(r.amount)}</span>
            </div>
          ))}
        </section>

        <section className="card stack" aria-labelledby="util-heading">
          <h3 id="util-heading" className="demo-h3">
            Technician utilization
          </h3>
          {REPORT.technicians.map((t) => (
            <div key={t.name} className="bar-row">
              <span>{t.name}</span>
              <span className="meter-track">
                <span className="meter-fill" style={{ width: `${t.utilization}%` }} />
              </span>
              <span className="right num">
                {t.utilization}% · {t.jobs} {t.jobs === 1 ? 'job' : 'jobs'}
              </span>
            </div>
          ))}
        </section>
      </div>

      <section className="card stack" aria-labelledby="materials-heading">
        <h3 id="materials-heading" className="demo-h3">
          Materials consumed
        </h3>
        <div className="scroll-x">
          <table className="demo-table">
            <thead>
              <tr>
                <th scope="col">Material</th>
                <th scope="col" className="right">
                  Used today
                </th>
                <th scope="col" className="right">
                  Billed
                </th>
                <th scope="col" className="right">
                  In stock
                </th>
                <th scope="col" className="right">
                  Status
                </th>
              </tr>
            </thead>
            <tbody>
              {REPORT.materials.map((m) => (
                <tr key={m.name}>
                  <td>{m.name}</td>
                  <td className="right">{m.used}</td>
                  <td className="right">{formatVnd(m.billed)}</td>
                  <td className="right">{m.stock}</td>
                  <td className="right">
                    <span className={`badge ${m.low ? 'badge-warn' : 'badge-active'}`}>
                      {m.low ? 'Reorder' : 'OK'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
