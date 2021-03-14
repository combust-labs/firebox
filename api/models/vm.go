// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// VM Virtual Machine
//
// swagger:model VM
type VM struct {

	// Virtual Machine ID.
	ID string `json:"id,omitempty"`

	// IP address of VM
	IP string `json:"ip,omitempty"`
}

// Validate validates this VM
func (m *VM) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this VM based on context it is used
func (m *VM) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *VM) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *VM) UnmarshalBinary(b []byte) error {
	var res VM
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
