package order

import (
	"time"
)

// #c.include #include <SystemConfig.h>
// #c.include <stdint.h>
// #ts.import import moment from "moment";

type Alias struct {
	Name string `ctype:"char[255]" validate:"presence,min=2,max=32"`
}

// some random comment

type User struct { // here we
	Id      int     `validate:"-"`
	Name    string  `ctype:"char[255]" validate:"presence,min=2,max=32"`
	Email   string  `ctype:"char[255]" validate:"email,required"`
	Value   float64 `validate:"presence,min=0"`
	Self    *Alias  // this just points back to ourself
	Aliases []Alias

	Time_of_day  time.Time
	Time_of_year time.Time `tstype:"string | moment.Moment"`

	ByteBuffer []byte
	Int8Buffer []int8
	Unknown    interface{}
	MapValues  map[int]string
} // another value

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
	OrderID       bid2.Bid2 `json:"orderID"` // bid2 is an unknown type
	InvoiceNumber int       `json:"invoiceNumber"`
	Name          string    `json:"name"`
	Active        bool      `json:"active" mysql:"__Active"`

	OrderStatus OrderStatus `json:"orderStatus"`

	OrderShippingOptions []ShippingOption `json:"orderShippingOptions,omitempty"`

	AnotherFile Another
}

type Nested struct {
	Values struct {
		Name string `ctype:"char[255]"`
	}
}
