package mel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type ErrorType uint64

const (
	ErrorTypeAny     ErrorType = (1 << 64) - 1
	ErrorTypeBind    ErrorType = 1 << 63 // used when c.Bind() fails
	ErrorTypeRender  ErrorType = 1 << 62 // used when c.Render() fails
	ErrorTypePrivate ErrorType = 1 << 0
	ErrorTypePublic  ErrorType = 1 << 1
)

type (
	Error struct {
		Err  error
		Type ErrorType
		Meta interface{}
	}

	errors []*Error
)

var _ error = &Error{}

func (msg *Error) JSON() interface{} {
	object := Object{}
	if msg.Meta != nil {
		value := reflect.ValueOf(msg.Meta)
		switch value.Kind() {
		case reflect.Struct:
			return msg.Meta
		case reflect.Map:
			for _, key := range value.MapKeys() {
				object[key.String()] = value.MapIndex(key).Interface()
			}
		default:
			object["meta"] = msg.Meta
		}
	}
	if _, ok := object["error"]; !ok {
		object["error"] = msg.Error()
	}
	return object
}

// MarshalJSON implements the json.Marshaller interface
func (msg *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(msg.JSON())
}

// Error implements the error interface
func (msg *Error) Error() string {
	return msg.Err.Error()
}

func (msg *Error) IsType(t ErrorType) bool {
	return (msg.Type & t) > 0
}

// ByType returns a readonly copy filtered by the type.
// Example, ByType(gin.ErrorTypePublic) returns a slice of errors with type=ErrorTypePublic
func (errs errors) ByType(t ErrorType) errors {
	if len(errs) == 0 {
		return nil
	}
	if t == ErrorTypeAny {
		return errs
	}
	var result errors
	for _, err := range errs {
		if err.IsType(t) {
			result = append(result, err)
		}
	}
	return result
}

// Last returns the last error in the errs.
// It returns nil if the errs is empty.
func (errs errors) Last() *Error {
	length := len(errs)
	if length > 0 {
		return errs[length-1]
	}
	return nil
}

// Errors returns a slice of all the error messages.
// Example:
// 		c.Error(errors.New("first"))
// 		c.Error(errors.New("second"))
// 		c.Error(errors.New("third"))
// 		c.Errors.Errors() // == []string{"first", "second", "third"}
func (errs errors) Errors() []string {
	if len(errs) == 0 {
		return nil
	}
	errStrs := make([]string, len(errs))
	for i, err := range errs {
		errStrs[i] = err.Error()
	}
	return errStrs
}

func (errs errors) JSON() interface{} {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs.Last().JSON()
	default:
		array := make(Array, len(errs))
		for i, err := range errs {
			array[i] = err.JSON()
		}
		return array
	}
}

func (errs errors) MarshalJSON() ([]byte, error) {
	return json.Marshal(errs.JSON())
}

func (errs errors) String() string {
	if len(errs) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	for i, err := range errs {
		fmt.Fprintf(&buffer, "Error #%02d: %s\n", (i + 1), err.Err)
		if err.Meta != nil {
			fmt.Fprintf(&buffer, "     Meta: %v\n", err.Meta)
		}
	}
	return buffer.String()
}
