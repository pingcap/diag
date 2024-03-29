// Code generated by go-swagger; DO NOT EDIT.

package types

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// UploadTask upload task
//
// swagger:model UploadTask
type UploadTask struct {

	// date
	Date string `json:"date,omitempty"`

	// id
	ID string `json:"id,omitempty"`

	// result
	Result string `json:"result,omitempty"`

	// status
	Status string `json:"status,omitempty"`
}

// Validate validates this upload task
func (m *UploadTask) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this upload task based on context it is used
func (m *UploadTask) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *UploadTask) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *UploadTask) UnmarshalBinary(b []byte) error {
	var res UploadTask
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
