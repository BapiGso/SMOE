package mymiddleware

import (
	"sync"

	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	once     sync.Once
	validate *validator.Validate
}

func (c *Validator) Validate(i any) error {
	c.once.Do(func() {
		c.validate = validator.New()
	})
	if err := defaults.Set(i); err != nil {
		return err
	}
	return c.validate.Struct(i)
}
