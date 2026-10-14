package entity

// Role is identity-service's user role, as carried in its access tokens.
type Role string

const (
	RoleCustomer   Role = "CUSTOMER"
	RoleTechnician Role = "TECHNICIAN"
	RoleManager    Role = "MANAGER"
	RoleAdmin      Role = "ADMIN"
)

// StaffRoles are the dealership's own users, as opposed to customers.
var StaffRoles = []Role{RoleTechnician, RoleManager, RoleAdmin}

// Principal is the caller a verified access token identifies. UserID is
// identity-service's user id, not this service's own ids.
type Principal struct {
	UserID string
	Role   Role
	// Token is the access token the caller sent, forwarded as-is when
	// scheduler asks customer-service and vehicle-service about them.
	Token string
}
