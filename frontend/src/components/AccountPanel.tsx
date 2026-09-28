import type { User } from '../api/identity'

const roleLabels: Record<User['role'], string> = {
  CUSTOMER: 'Customer',
  TECHNICIAN: 'Technician',
  MANAGER: 'Dealership manager',
  ADMIN: 'Administrator',
}

export function AccountPanel({ user }: { user: User }) {
  return (
    <section className="panel narrow stack" aria-labelledby="account-heading">
      <h2 id="account-heading">Account</h2>
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
  )
}
