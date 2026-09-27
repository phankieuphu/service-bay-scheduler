import { useState } from 'react'
import type { User } from '../api/identity'
import { useAuth } from '../auth/auth-context'

export function UserMenu({ user }: { user: User }) {
  const { signOut } = useAuth()
  const [signingOut, setSigningOut] = useState(false)

  return (
    <div className="user-menu">
      <span className="user-email" title={user.email}>
        {user.email}
      </span>
      <span className="badge badge-role">{user.role}</span>
      <button
        type="button"
        className="secondary"
        disabled={signingOut}
        onClick={() => {
          setSigningOut(true)
          void signOut()
        }}
      >
        Sign out
      </button>
    </div>
  )
}
