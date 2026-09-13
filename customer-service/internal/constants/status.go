package constants

type CustomerStatus string

const (
	StatusActive   CustomerStatus = "ACTIVE"
	StatusInactive CustomerStatus = "INACTIVE"
	StatusBanned   CustomerStatus = "BANNED"
)
