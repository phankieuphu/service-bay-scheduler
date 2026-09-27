package constants

type Role string

const (
	RoleCustomer   Role = "CUSTOMER"
	RoleTechnician Role = "TECHNICIAN"
	RoleManager    Role = "MANAGER"
	RoleAdmin      Role = "ADMIN"
)

// SelfRegisterable reports whether POST /auth/register may create a user
// with this role. MANAGER and ADMIN accounts grant control over other
// users' data, so they must be provisioned by an existing admin instead.
func (r Role) SelfRegisterable() bool {
	return r == RoleCustomer || r == RoleTechnician
}

type UserStatus string

const (
	UserActive   UserStatus = "ACTIVE"
	UserInactive UserStatus = "INACTIVE"
	UserLocked   UserStatus = "LOCKED"
)
