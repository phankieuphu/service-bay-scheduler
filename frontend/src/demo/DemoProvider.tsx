import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { DemoContext, type DemoBooking, type DemoContextValue, type DemoState } from './demo-context'

const initialState: DemoState = {
  booking: null,
  stage: 'scheduled',
  changeDecision: null,
  paidMethod: null,
  events: [],
}

let nextEventId = 1

// DemoProvider holds the simulated booking → service → bill flow in memory.
// Each action appends the event the owning service would publish (topics per
// docs/architecture-design.md §3), so the overview can show the stream.
export function DemoProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<DemoState>(initialState)

  const emit = useCallback((s: DemoState, events: [string, string, string][]): DemoState => {
    const added = events.map(([type, topic, key]) => ({
      id: nextEventId++,
      type,
      topic,
      key,
      at: new Date(),
    }))
    return { ...s, events: [...added.reverse(), ...s.events] }
  }, [])

  const value = useMemo<DemoContextValue>(() => {
    const bookingKey = (s: DemoState) => String(s.booking?.id ?? '')
    return {
      state,
      confirmBooking: (booking: Omit<DemoBooking, 'id'>) =>
        setState((s) => {
          const full = { ...booking, id: 500 + s.events.length }
          const next: DemoState = { ...initialState, events: s.events, booking: full }
          const events: [string, string, string][] = [
            ['BookingConfirmed', 'scheduler.booking.confirmed', String(full.id)],
          ]
          if (full.deposit > 0) {
            events.push(['PaymentReceived', 'billing.payment.received', String(full.id)])
          }
          return emit(next, events)
        }),
      startService: () => setState((s) => ({ ...s, stage: 'in-service' })),
      requestChange: () =>
        setState((s) =>
          emit({ ...s, stage: 'waiting' }, [
            ['MidServiceApprovalRequested', 'scheduler.booking.approval-requested', bookingKey(s)],
          ]),
        ),
      answerChange: (approved: boolean) =>
        setState((s) => ({
          ...s,
          stage: 'in-service',
          changeDecision: approved ? 'approved' : 'declined',
        })),
      completeService: () =>
        setState((s) =>
          emit({ ...s, stage: 'completed' }, [
            ['ServiceCompleted', 'scheduler.appointment.service-completed.v1', bookingKey(s)],
            ['MaterialsConsumed', 'dealership.materials.consumed', bookingKey(s)],
          ]),
        ),
      payBalance: (method: string) =>
        setState((s) =>
          emit({ ...s, paidMethod: method }, [
            ['PaymentReceived', 'billing.payment.received', bookingKey(s)],
          ]),
        ),
      reset: () => setState(initialState),
    }
  }, [state, emit])

  return <DemoContext.Provider value={value}>{children}</DemoContext.Provider>
}
