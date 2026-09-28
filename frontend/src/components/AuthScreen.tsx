import { useState } from 'react'
import { useAuth } from '../auth/auth-context'
import { SignInForm } from './SignInForm'
import { SignUpForm } from './SignUpForm'
import { Tabs } from './Tabs'

type Mode = 'sign-in' | 'sign-up'

export function AuthScreen() {
  const { state } = useAuth()
  const [mode, setMode] = useState<Mode>('sign-in')
  const notice = state.status === 'signed-out' ? state.notice : undefined

  return (
    <div className="panel auth-panel stack">
      <Tabs
        className="subtabs"
        tabs={[
          { id: 'sign-in', label: 'Sign in' },
          { id: 'sign-up', label: 'Create account' },
        ]}
        active={mode}
        onChange={setMode}
      />
      {notice && <p className="message info">{notice}</p>}
      {mode === 'sign-in' ? <SignInForm /> : <SignUpForm />}
    </div>
  )
}
