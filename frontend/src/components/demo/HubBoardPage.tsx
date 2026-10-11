import { useState } from 'react'
import {
  CHANGE_REQUEST,
  JOB_STATE_LABELS,
  MATERIALS_BY_SERVICE,
  OTHER_JOBS,
  TECHNICIAN,
  formatVnd,
  type BayJob,
  type JobState,
} from '../../demo/data'
import { useDemo, type ServiceStage } from '../../demo/demo-context'
import type { DemoSection } from './sections'

const HOURS = [8, 9, 10, 11, 12, 13, 14, 15, 16, 17]
const DAY_START = 8
const DAY_LENGTH = 10
const MY_BAY = 'Bay 2'

const STAGE_TO_JOB: Record<ServiceStage, JobState> = {
  scheduled: 'confirmed',
  'in-service': 'in-service',
  waiting: 'waiting',
  completed: 'completed',
}

const PROGRESS = ['Received', 'Assigned', 'Bay ready', 'Car in', 'In service', 'Completed']

function parseTime(time: string): number {
  const [h, m] = time.split(':').map(Number)
  return h + m / 60
}

export function HubBoardPage({ onNavigate }: { onNavigate: (section: DemoSection) => void }) {
  const { state, startService, requestChange, answerChange, completeService } = useDemo()
  const { booking, stage, changeDecision } = state
  const [note, setNote] = useState(CHANGE_REQUEST.note)

  const bays = Object.entries(OTHER_JOBS).map(([name, jobs]) => {
    const all: BayJob[] = [...jobs]
    if (booking && name === MY_BAY) {
      const start = parseTime(booking.time)
      const minutes = booking.services.reduce((a, s) => a + s.minutes, 0)
      all.push({
        plate: booking.plate,
        what: booking.services.map((s) => s.name).join(' · '),
        start,
        end: Math.min(DAY_START + DAY_LENGTH, start + Math.max(minutes, 30) / 60),
        state: STAGE_TO_JOB[stage],
        mine: true,
      })
    }
    return { name, jobs: all }
  })

  // Steps 0–3 are done once the booking exists; "In service" is current until
  // the technician completes it.
  const progressDone = !booking ? 0 : stage === 'scheduled' ? 3 : stage === 'completed' ? 6 : 4
  const materials = booking
    ? booking.services.flatMap((s) => MATERIALS_BY_SERVICE[s.id] ?? [])
    : []
  const legend: JobState[] = ['confirmed', 'held', 'in-service', 'waiting', 'completed']

  return (
    <div className="stack roomy demo">
      <div className="section-head wrap">
        <div>
          <h2 className="demo-title">{booking ? `${booking.hub.name} · ${booking.dayLabel}` : 'District 7 Hub · Today'}</h2>
          <p className="muted flush">4 bays · 6 technicians on shift · sample schedule</p>
        </div>
      </div>

      <section className="card stack" aria-labelledby="bays-heading">
        <div className="section-head wrap">
          <h3 id="bays-heading" className="demo-h3">
            Bays
          </h3>
          <ul className="legend">
            {legend.map((s) => (
              <li key={s}>
                <span className={`swatch job-${s}`} /> {JOB_STATE_LABELS[s]}
              </li>
            ))}
          </ul>
        </div>
        <div className="scroll-x">
          <div className="timeline">
            <div className="timeline-row">
              <span />
              <div className="timeline-hours">
                {HOURS.map((h) => (
                  <span key={h} className="small muted">
                    {String(h).padStart(2, '0')}
                  </span>
                ))}
              </div>
            </div>
            {bays.map((bay) => (
              <div key={bay.name} className="timeline-row">
                <strong>{bay.name}</strong>
                <div className="timeline-track">
                  {bay.jobs.map((j) => (
                    <div
                      key={`${j.plate}-${j.start}`}
                      className={`job job-${j.state}${j.mine ? ' mine' : ''}`}
                      style={{
                        left: `${((j.start - DAY_START) / DAY_LENGTH) * 100}%`,
                        width: `${((j.end - j.start) / DAY_LENGTH) * 100}%`,
                      }}
                      title={`${j.plate} · ${j.what} · ${JOB_STATE_LABELS[j.state]}`}
                    >
                      <strong>{j.plate}</strong>
                      <span>{j.what}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {!booking ? (
        <section className="card placeholder">
          <div className="stack">
            <p className="muted">No demo booking yet. Book a slot to see it arrive on this board.</p>
            <div>
              <button type="button" className="primary" onClick={() => onNavigate('booking')}>
                Book a service
              </button>
            </div>
          </div>
        </section>
      ) : (
        <div className="split">
          <section className="split-main card stack focus-card" aria-labelledby="booking-heading">
            <div className="section-head wrap">
              <div>
                <h3 id="booking-heading" className="demo-title small-title">
                  {booking.plate}
                </h3>
                <p className="muted small flush">
                  Booking #{booking.id} · {MY_BAY} · {booking.dayLabel} {booking.time}
                </p>
              </div>
              <span className={`badge job-badge job-${STAGE_TO_JOB[stage]}`}>
                {stage === 'waiting' ? 'Waiting for customer approval' : JOB_STATE_LABELS[STAGE_TO_JOB[stage]]}
              </span>
            </div>
            <ol className="progress" aria-label="Progress">
              {PROGRESS.map((p, i) => (
                <li
                  key={p}
                  className={i < progressDone ? 'done' : i === progressDone ? 'current' : ''}
                >
                  <span className="bar" />
                  {p}
                </li>
              ))}
            </ol>
            <div className="two-col">
              <div className="stack">
                <h4 className="demo-h4">Technician</h4>
                <strong>{TECHNICIAN.name}</strong>
                {TECHNICIAN.skills.map((k) => (
                  <div key={k.name} className="meter">
                    <span className="section-head small">
                      <span>{k.name}</span>
                      <span className="num">
                        L{k.level.toFixed(1)} · {k.count}/10 to next
                      </span>
                    </span>
                    <span className="meter-track">
                      <span className="meter-fill" style={{ width: `${k.count * 10}%` }} />
                    </span>
                  </div>
                ))}
              </div>
              <div className="stack">
                <h4 className="demo-h4">Materials prepared</h4>
                {materials.length === 0 ? (
                  <p className="muted small flush">None needed.</p>
                ) : (
                  <ul className="plain-list small">
                    {materials.map((m) => (
                      <li key={m.name} className="section-head">
                        <span>{m.name}</span>
                        <span>
                          {m.qty} {m.unit}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
            <div className="actions">
              {stage === 'scheduled' && (
                <button type="button" className="primary" onClick={startService}>
                  Receive vehicle &amp; start service
                </button>
              )}
              {stage === 'in-service' && (
                <button type="button" className="primary" onClick={completeService}>
                  Mark service complete
                </button>
              )}
              {stage === 'completed' && (
                <>
                  <span className="message success flush">Service completed.</span>
                  <button type="button" className="secondary" onClick={() => onNavigate('billing')}>
                    View the bill →
                  </button>
                </>
              )}
            </div>
          </section>

          <aside className="split-side panel stack" aria-labelledby="change-heading">
            <h3 id="change-heading" className="demo-h3">
              Mid-service change
            </h3>
            <p className="small muted flush">
              Found during work. The customer approves or declines on their phone; the booking
              continues either way.
            </p>
            <div className="card tight stack small">
              <span className="section-head">
                <span>{CHANGE_REQUEST.service}</span>
                <strong>{formatVnd(CHANGE_REQUEST.price)}</strong>
              </span>
              <span className="section-head muted">
                <span>{CHANGE_REQUEST.material.name} × 1</span>
                <span>{formatVnd(CHANGE_REQUEST.material.unitPrice)}</span>
              </span>
              <span className="section-head muted">
                <span>Extra time</span>
                <span>+{CHANGE_REQUEST.extraMinutes} min</span>
              </span>
            </div>
            <label className="field">
              <span className="label">Note to customer</span>
              <textarea
                rows={3}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                disabled={stage !== 'in-service' || changeDecision !== null}
              />
            </label>
            {stage === 'waiting' && (
              <>
                <p className="message flush warn-box">Sent · waiting for the customer’s reply</p>
                <p className="small muted flush">Demo: answer as the customer.</p>
                <div className="actions">
                  <button type="button" className="secondary" onClick={() => answerChange(false)}>
                    Decline
                  </button>
                  <button type="button" className="primary" onClick={() => answerChange(true)}>
                    Approve
                  </button>
                </div>
              </>
            )}
            {changeDecision && (
              <p className={`message flush ${changeDecision === 'approved' ? 'success' : 'info'}`}>
                Customer {changeDecision} the extra work.
              </p>
            )}
            {stage === 'in-service' && changeDecision === null && (
              <button type="button" className="primary" onClick={requestChange}>
                Request approval
              </button>
            )}
            {stage === 'scheduled' && (
              <p className="small muted flush">Available once the service has started.</p>
            )}
          </aside>
        </div>
      )}
    </div>
  )
}
