import type { User } from '../api/identity'

const roleLabels: Record<User['role'], string> = {
  CUSTOMER: 'Customer',
  TECHNICIAN: 'Technician',
  MANAGER: 'Dealership manager',
  ADMIN: 'Administrator',
}

export function AccountPanel({ user }: { user: User }) {
  return (
    <div className="panel">
      <h2>Account</h2>
      <table className="details">
        <tbody>
          <tr>
            <th>Email</th>
            <td>{user.email}</td>
          </tr>
          <tr>
            <th>Role</th>
            <td>{roleLabels[user.role]}</td>
          </tr>
          <tr>
            <th>Status</th>
            <td>
              <span className={`badge badge-${user.status.toLowerCase()}`}>{user.status}</span>
            </td>
          </tr>
          <tr>
            <th>User ID</th>
            <td>
              <code>{user.id}</code>
            </td>
          </tr>
          <tr>
            <th>Member since</th>
            <td>{new Date(user.created_at).toLocaleString()}</td>
          </tr>
        </tbody>
      </table>
    </div>
  )
}
