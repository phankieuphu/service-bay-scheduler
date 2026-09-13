package ports

import "customer-service/internal/domain/entity"

// ListCustomersParams is a cursor-based pagination request. Cursor is the ID
// of the last customer the caller has already seen (0 to start from the
// beginning); results are ordered by ID ascending and only IDs greater than
// Cursor are returned.
type ListCustomersParams struct {
	Cursor int64
	Limit  int
}

// CustomerPage is a single page of a cursor-paginated customer listing.
// NextCursor is only meaningful when HasMore is true.
type CustomerPage struct {
	Customers  []entity.Customer
	NextCursor int64
	HasMore    bool
}
