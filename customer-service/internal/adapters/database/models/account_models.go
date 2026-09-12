package models

type Customer struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (a Customer) TableName() string {
	return "customer"
}
