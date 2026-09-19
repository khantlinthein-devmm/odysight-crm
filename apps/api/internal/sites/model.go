package sites

import "time"

// PropertyType describes what kind of place a site is. Commercial cleaning
// covers more than the residential set customers were limited to.
type PropertyType string

const (
	PropertyHouse      PropertyType = "house"
	PropertyCondo      PropertyType = "condo"
	PropertyApartment  PropertyType = "apartment"
	PropertyOffice     PropertyType = "office"
	PropertyRestaurant PropertyType = "restaurant"
	PropertyFactory    PropertyType = "factory"
	PropertyMall       PropertyType = "mall"
	PropertyRetail     PropertyType = "retail"
	PropertyOther      PropertyType = "other"
)

func (p PropertyType) Valid() bool {
	switch p {
	case PropertyHouse, PropertyCondo, PropertyApartment, PropertyOffice,
		PropertyRestaurant, PropertyFactory, PropertyMall, PropertyRetail, PropertyOther:
		return true
	}
	return false
}

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

func (s Status) Valid() bool {
	return s == StatusActive || s == StatusInactive
}

// Site is a physical place a customer has cleaned: a branch, a building, or a
// single common area. The commercial relationship stays on the customer.
type Site struct {
	ID           int64
	CustomerID   int64
	CustomerName string
	Name         string
	Address      string
	Area         string
	PropertyType PropertyType
	// Contact fields are per-site overrides. Blank means the customer's own
	// contact details apply.
	ContactName  string
	ContactPhone string
	ContactEmail string
	Notes        string
	Status       Status
	Lat          *float64
	Lng          *float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
