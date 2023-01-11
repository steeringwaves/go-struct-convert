package order

import (
	"time"
)

type Alias struct {
	Name string `ctype:"char[255]" validate:"presence,min=2,max=32"`
}

type User struct {
	Id      int     `validate:"-"`
	Name    string  `ctype:"char[255]" validate:"presence,min=2,max=32"`
	Email   string  `ctype:"char[255]" validate:"email,required"`
	Value   float64 `validate:"presence,min=0"`
	Self    *Alias
	Aliases []Alias

	Time_of_day time.Time

	ByteBuffer []byte
	Int8Buffer []int8
	Unknown    interface{}
}

type OrderStatus struct {
	OrderStatusID int    `json:"orderStatusID"`
	Name          string `json:"name"`
}

type State struct {
	StateID   int    `json:"stateID"`
	Name      string `json:"name"`
	StateCode string `json:"stateCode"`
}

type ShippingOption struct {
	ShippingOptionID uint   `json:"shippingOptionID"`
	FullName         string `json:"fullName"`
	Company          string `json:"company"`
	Email            string `json:"email"`
	AddressLine1     string `json:"addressLine1"`
	AddressLine2     string `json:"addressLine2"`
	City             string `json:"city"`
	State            State  `json:"state,omitempty"`
	Zip              string `json:"zip"`
	Phone            string `json:"phone"`

	TaxExemptCertificateNumber string `json:"taxExemptCertificateNumber"`
}

var order struct {
	OrderID       bid2.Bid2 `json:"orderID"`
	InvoiceNumber int       `json:"invoiceNumber"`
	Name          string    `json:"name"`
	Active        bool      `json:"active" mysql:"__Active"`

	OrderStatus OrderStatus `json:"orderStatus"`

	OrderShippingOptions []ShippingOption `json:"orderShippingOptions,omitempty"`
}
