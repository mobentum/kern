package xvalidator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

type createUserRequest struct {
	Name     string `validate:"required,min=3,max=50"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"gte=18,lte=120"`
	Role     string `validate:"required,oneof=admin user moderator"`
	Password string `validate:"required,min=8"`
}

func TestValidateSuccess(t *testing.T) {
	req := createUserRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Age:      30,
		Role:     "admin",
		Password: "securepass123",
	}

	err := Validate(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateFailure(t *testing.T) {
	req := createUserRequest{
		Name:     "ab",
		Email:    "not-an-email",
		Age:      10,
		Role:     "superadmin",
		Password: "short",
	}

	err := Validate(req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	ve, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	fields := make(map[string]string)
	for _, e := range ve {
		fields[e.Field] = e.Tag
	}

	expected := map[string]string{
		"Name":     "min",
		"Email":    "email",
		"Age":      "gte",
		"Role":     "oneof",
		"Password": "min",
	}

	for field, tag := range expected {
		if got, ok := fields[field]; !ok {
			t.Errorf("missing error for field %s", field)
		} else if got != tag {
			t.Errorf("field %s: expected tag %s, got %s", field, tag, got)
		}
	}

	for _, e := range ve {
		if e.Field == "Name" && e.Message != "must be at least 3" {
			t.Errorf("Name: expected default min message, got %q", e.Message)
		}
		if e.Field == "Email" && e.Message != "must be a valid email address" {
			t.Errorf("Email: expected email message, got %q", e.Message)
		}
	}
}

func TestCustomMessages(t *testing.T) {
	req := createUserRequest{
		Name:  "ab",
		Email: "bad",
		Age:   10,
	}

	messages := map[string]string{
		"min":   "is too short (minimum: %s)",
		"email": "provide a valid email address",
		"gte":   "must be at least %s",
	}

	err := ValidateWithMessages(req, messages)
	if err == nil {
		t.Fatal("expected validation error")
	}

	ve := err.(ValidationErrors)
	for _, e := range ve {
		if e.Field == "Name" && e.Message != "is too short (minimum: 3)" {
			t.Errorf("Name: expected custom message, got %q", e.Message)
		}
		if e.Field == "Email" && e.Message != "provide a valid email address" {
			t.Errorf("Email: expected custom message, got %q", e.Message)
		}
	}
}

func TestCustomValidationRule(t *testing.T) {
	v := New()

	err := v.RegisterValidation("even", func(fl validator.FieldLevel) bool {
		n, ok := fl.Field().Interface().(int)
		if !ok {
			return false
		}
		return n%2 == 0
	})
	if err != nil {
		t.Fatalf("register validation: %v", err)
	}

	type evenRequest struct {
		Value int `validate:"even"`
	}

	err = v.Validate(evenRequest{Value: 3})
	if err == nil {
		t.Fatal("expected validation error for odd value")
	}

	err = v.Validate(evenRequest{Value: 4})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStructLevelValidation(t *testing.T) {
	v := New()

	type passwordRequest struct {
		Password        string `validate:"required,min=8"`
		ConfirmPassword string `validate:"required"`
	}

	v.RegisterStructValidation(func(sl validator.StructLevel) {
		r := sl.Current().Interface().(passwordRequest)
		if r.Password != r.ConfirmPassword {
			sl.ReportError(r.ConfirmPassword, "ConfirmPassword", "ConfirmPassword", "eqfield", "Password")
		}
	}, passwordRequest{})

	v.SetDefaultMessage("eqfield", "must match Password")

	err := v.Validate(passwordRequest{
		Password:        "strongpass123",
		ConfirmPassword: "differentpass",
	})
	if err == nil {
		t.Fatal("expected struct-level validation error")
	}

	ve := err.(ValidationErrors)
	found := false
	for _, e := range ve {
		if e.Field == "ConfirmPassword" && e.Tag == "eqfield" {
			found = true
			if e.Message != "must match Password" {
				t.Errorf("expected custom message, got %q", e.Message)
			}
		}
	}
	if !found {
		t.Fatal("expected ConfirmPassword eqfield error")
	}
}
