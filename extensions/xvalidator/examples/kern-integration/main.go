package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/mobentum/kern"
	"github.com/mobentum/kern/extensions/xvalidator"
)

type createUserRequest struct {
	Name     string `json:"name"     validate:"required,min=3,max=50"`
	Email    string `json:"email"    validate:"required,email"`
	Age      int    `json:"age"      validate:"gte=18,lte=120"`
	Role     string `json:"role"     validate:"required,oneof=admin user moderator"`
	Password string `json:"password" validate:"required,min=8"`
}

type updateUserRequest struct {
	Name  string `json:"name"  validate:"omitempty,min=3,max=50"`
	Email string `json:"email" validate:"omitempty,email"`
	Age   int    `json:"age"   validate:"omitempty,gte=18,lte=120"`
}

type configRequest struct {
	Host      string `json:"host"      validate:"required,hostname"`
	Port      int    `json:"port"      validate:"required,min=1,max=65535"`
	Timeout   int    `json:"timeout"   validate:"required,min=100,max=30000"`
	ThemeColor string `json:"theme_color" validate:"omitempty,hex_color"`
	Secure    bool   `json:"secure"`
}

type createOrderRequest struct {
	UserID  string          `json:"user_id"  validate:"required,uuid4"`
	Items   []orderItem     `json:"items"    validate:"required,min=1,dive"`
	Billing billingInfo     `json:"billing"  validate:"required"`
	Notes   string          `json:"notes"    validate:"omitempty,max=500"`
}

type orderItem struct {
	ProductID string  `json:"product_id" validate:"required,uuid4"`
	Quantity  int     `json:"quantity"   validate:"required,min=1,max=100"`
	Price     float64 `json:"price"      validate:"required,gt=0"`
}

type billingInfo struct {
	Address string `json:"address" validate:"required,min=10,max=200"`
	City    string `json:"city"    validate:"required"`
	ZIP     string `json:"zip"     validate:"required,len=5,numeric"`
}

type dateRangeRequest struct {
	StartDate string `json:"start_date" validate:"required,datetime=2006-01-02"`
	EndDate   string `json:"end_date"   validate:"required,datetime=2006-01-02,gtefield=StartDate"`
}

func main() {
	app := kern.New()

	registerCustomValidation()
	setupCustomMessages()

	app.POST("/users", handleCreateUser)
	app.PUT("/users/{id}", handleUpdateUser)
	app.POST("/config", handleConfig)
	app.POST("/orders", handleCreateOrder)
	app.POST("/date-range", handleDateRange)
	app.POST("/register", handleRegisterWithStructLevel)

	log.Fatal(app.Run(":8080"))
}

func registerCustomValidation() {
	err := xvalidator.RegisterValidation("hex_color", func(fl validator.FieldLevel) bool {
		v := fl.Field().String()
		if len(v) != 7 && len(v) != 4 {
			return false
		}
		if v[0] != '#' {
			return false
		}
		for _, c := range v[1:] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	})
	if err != nil {
		log.Printf("register validation: %v", err)
	}

	xvalidator.RegisterStructValidation(func(sl validator.StructLevel) {
		r := sl.Current().Interface().(createOrderRequest)
		total := 0.0
		for _, item := range r.Items {
			total += item.Price * float64(item.Quantity)
		}
		if total > 10000 {
			sl.ReportError(r.Items, "Items", "Items", "max_total", "10000")
		}
	}, createOrderRequest{})
}

func setupCustomMessages() {
	xvalidator.Default().SetMessages(map[string]string{
		"required": "is required and cannot be empty",
		"min":      "must be at least %s characters",
		"max":      "must not exceed %s characters",
		"email":    "must be a valid email address",
		"uuid4":    "must be a valid UUID v4",
		"gte":      "must be at least %s",
		"lte":      "must be at most %s",
		"oneof":    "must be one of: admin, user, moderator",
		"hex_color": "must be a valid hex color (e.g. #ff0000 or #f00)",
	})
}

func handleCreateUser(c *kern.Context) {
	var req createUserRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := xvalidator.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusCreated, map[string]string{
		"message": fmt.Sprintf("user %s created", req.Name),
	})
}

func handleUpdateUser(c *kern.Context) {
	var req updateUserRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := xvalidator.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("user %s updated", c.Param("id")),
	})
}

func handleConfig(c *kern.Context) {
	var req configRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := xvalidator.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusOK, map[string]string{"status": "config applied"})
}

func handleCreateOrder(c *kern.Context) {
	var req createOrderRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := xvalidator.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusCreated, map[string]string{
		"message": "order created",
	})
}

func handleDateRange(c *kern.Context) {
	var req dateRangeRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := xvalidator.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusOK, map[string]string{"status": "valid date range"})
}

type registerRequest struct {
	Username        string `validate:"required,min=3,max=30,alphanum"`
	Email           string `validate:"required,email"`
	Password        string `validate:"required,min=8,max=64"`
	ConfirmPassword string `validate:"required"`
}

func handleRegisterWithStructLevel(c *kern.Context) {
	v := xvalidator.New()

	v.RegisterStructValidation(func(sl validator.StructLevel) {
		r := sl.Current().Interface().(registerRequest)

		if r.Password != r.ConfirmPassword {
			sl.ReportError(r.ConfirmPassword, "ConfirmPassword", "ConfirmPassword", "eqfield", "Password")
		}
	}, registerRequest{})

	v.SetDefaultMessage("eqfield", "must match Password")

	var req registerRequest
	if err := c.DecodeJSON(&req); err != nil {
		_ = c.Error(http.StatusBadRequest, "invalid request body")
		return
	}

	if err := v.Validate(req); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"error":  "validation failed",
			"fields": err,
		})
		return
	}

	_ = c.JSON(http.StatusCreated, map[string]string{"message": "registered"})
}
