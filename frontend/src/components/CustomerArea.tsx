import { useEffect, useState } from 'react'
import { ApiError, getMyProfile, type Customer } from '../api/customer'
import { BookingPage } from './BookingPage'
import { MyAppointments } from './MyAppointments'

export type CustomerSection = 'book' | 'appointments'

interface Props {
  section: CustomerSection
  onNavigate: (section: CustomerSection) => void
}

type ProfileState =
  | { status: 'loading' }
  | { status: 'ready'; customer: Customer }
  | { status: 'missing' }
  | { status: 'error'; message: string }

// The booking screens need the signed-in user's customer profile (its id is
// what vehicles and appointments hang off). customer-service links one a
// moment after sign-up, so a 404 means "not yet" and offers a retry.
export function CustomerArea({ section, onNavigate }: Props) {
  const [profile, setProfile] = useState<ProfileState>({ status: 'loading' })
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    let cancelled = false
    getMyProfile()
      .then((customer) => {
        if (!cancelled) setProfile({ status: 'ready', customer })
      })
      .catch((err) => {
        if (cancelled) return
        if (err instanceof ApiError && err.status === 404) setProfile({ status: 'missing' })
        else setProfile({ status: 'error', message: err instanceof Error ? err.message : 'Failed to load your profile.' })
      })
    return () => {
      cancelled = true
    }
  }, [attempt])

  function retry() {
    setProfile({ status: 'loading' })
    setAttempt((n) => n + 1)
  }

  switch (profile.status) {
    case 'loading':
      return <p className="muted">Loading your profile…</p>
    case 'missing':
    case 'error':
      return (
        <div className="message info confirm">
          <span>
            {profile.status === 'missing'
              ? 'Your customer profile is still being set up. This takes a few seconds after signing up.'
              : `Couldn’t load your profile: ${profile.message}`}
          </span>
          <button type="button" className="secondary" onClick={retry}>
            Try again
          </button>
        </div>
      )
  }

  return section === 'book' ? (
    <BookingPage customer={profile.customer} onViewAppointments={() => onNavigate('appointments')} />
  ) : (
    <MyAppointments onBook={() => onNavigate('book')} />
  )
}
