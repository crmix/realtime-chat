package validator

import "github.com/go-playground/validator/v10"

// Validator wraps go-playground/validator with a singleton instance so that
// struct-tag caching is shared across the application.
type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	return &Validator{v: validator.New(validator.WithRequiredStructEnabled())}
}

func (vd *Validator) Struct(s any) error {
	return vd.v.Struct(s)
}
