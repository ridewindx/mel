package binding

import (
	"reflect"
	"sync"

	"gopkg.in/go-playground/validator.v9"
)

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ StructValidator = &defaultValidator{}

func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	kind := value.Kind()
	if kind == reflect.Ptr {
		kind = value.Elem().Kind()
	}
	return kind
}
