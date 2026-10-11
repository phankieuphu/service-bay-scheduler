import type { User } from '../api/identity'
import { PageHeader } from './PageHeader'

const roleLabels: Record<User['role'], string> = {
  CUSTOMER: 'Customer',
  TECHNICIAN: 'Technician',
  MANAGER: 'Dealership manager',
  ADMIN: 'Administrator',
}

export function AccountPanel({ user }: { user: User }) {
  return (
    <div className="stack roomy">
      <PageHeader
        title="Account"
        description="Your sign-in details, as identity-service holds them."
      />
      <section className="card narrow stack" aria-labelledby="account-heading">
        <h2 id="account-heading">Profile</h2>
        <dl className="details">
          <dt>Email</dt>
          <dd>{user.email}</dd>
          <dt>Role</dt>
          <dd>{roleLabels[user.role]}</dd>
          <dt>Status</dt>
          <dd>
            <span className={`badge badge-${user.status.toLowerCase()}`}>{user.status}</span>
          </dd>
          <dt>User ID</dt>
          <dd>
            <code>{user.id}</code>
          </dd>
          <dt>Member since</dt>
          <dd>{new Date(user.created_at).toLocaleString()}</dd>
        </dl>
      </section>
    </div>
  )
}
