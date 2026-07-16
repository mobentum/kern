package xvalidator

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e ValidationError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("%s: failed on %s", e.Field, e.Tag)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	msgs := make([]string, 0, len(e))
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

type Validator struct {
	v       *validator.Validate
	mu      sync.RWMutex
	msgMap  map[string]string
}

func New() *Validator {
	return &Validator{
		v:      validator.New(validator.WithRequiredStructEnabled()),
		msgMap: make(map[string]string),
	}
}

func (w *Validator) RegisterValidation(name string, fn validator.Func, callValidationEvenIfNull ...bool) error {
	return w.v.RegisterValidation(name, fn, callValidationEvenIfNull...)
}

func (w *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	w.v.RegisterStructValidation(fn, types...)
}

func (w *Validator) SetDefaultMessage(tag, message string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgMap[tag] = message
}

func (w *Validator) SetMessages(messages map[string]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for k, v := range messages {
		w.msgMap[k] = v
	}
}

func (w *Validator) Validate(data interface{}) error {
	err := w.v.Struct(data)
	if err == nil {
		return nil
	}

	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	errs := make(ValidationErrors, 0, len(ve))
	for _, fe := range ve {
		field := fe.Field()
		tag := fe.Tag()
		msg := w.msgMap[tag]
		if msg == "" {
			msg = defaultMessage(tag, fe.Param())
		} else if strings.Contains(msg, "%s") {
			msg = fmt.Sprintf(msg, fe.Param())
		}

		errs = append(errs, ValidationError{
			Field:   field,
			Tag:     tag,
			Value:   fe.Value(),
			Message: msg,
		})
	}
	return errs
}

var defaultValidator *Validator
var once sync.Once

func Default() *Validator {
	once.Do(func() {
		defaultValidator = New()
	})
	return defaultValidator
}

func Validate(data interface{}) error {
	return Default().Validate(data)
}

func ValidateWithMessages(data interface{}, messages map[string]string) error {
	v := Default()
	v.mu.Lock()
	for k, msg := range messages {
		v.msgMap[k] = msg
	}
	v.mu.Unlock()
	return v.Validate(data)
}

func RegisterValidation(name string, fn validator.Func, callValidationEvenIfNull ...bool) error {
	return Default().RegisterValidation(name, fn, callValidationEvenIfNull...)
}

func RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	Default().RegisterStructValidation(fn, types...)
}

func defaultMessage(tag, param string) string {
	switch tag {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", param)
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", param)
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "uuid4":
		return "must be a valid UUID v4"
	case "gt":
		return fmt.Sprintf("must be greater than %s", param)
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", param)
	case "lt":
		return fmt.Sprintf("must be less than %s", param)
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", param)
	case "eq":
		return fmt.Sprintf("must be equal to %s", param)
	case "ne":
		return fmt.Sprintf("must not be equal to %s", param)
	case "alpha":
		return "must contain only alphabetic characters"
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "numeric":
		return "must be numeric"
	case "ip":
		return "must be a valid IP address"
	case "ipv4":
		return "must be a valid IPv4 address"
	case "ipv6":
		return "must be a valid IPv6 address"
	case "datetime":
		return fmt.Sprintf("must be a valid datetime in format %s", param)
	default:
		return fmt.Sprintf("failed on %s", tag)
	}
}
