package constants

type Role string

const (
	RoleCustomer   Role = "CUSTOMER"
	RoleTechnician Role = "TECHNICIAN"
	RoleManager    Role = "MANAGER"
	RoleAdmin      Role = "ADMIN"
)

// SelfRegisterable reports whether POST /auth/register may create a user
// with this role. Only CUSTOMER: TECHNICIAN, MANAGER and ADMIN are staff
// roles that other services grant access to, so an admin provisions them
// (POST /admin/users).
func (r Role) SelfRegisterable() bool {
	return r == RoleCustomer
}

func (r Role) Valid() bool {
	switch r {
	case RoleCustomer, RoleTechnician, RoleManager, RoleAdmin:
		return true
	}
	return false
}

type UserStatus string

const (
	UserActive   UserStatus = "ACTIVE"
	UserInactive UserStatus = "INACTIVE"
	UserLocked   UserStatus = "LOCKED"
)

func (s UserStatus) Valid() bool {
	return s == UserActive || s == UserInactive || s == UserLocked
}
