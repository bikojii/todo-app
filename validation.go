package todo

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func ValidateText(value, field string, required bool) error {
	if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%w: %s contains invalid characters", ErrInvalidInput, field)
	}
	if utf8.RuneCountInString(value) > 255 {
		return fmt.Errorf("%w: %s exceeds 255 characters", ErrInvalidInput, field)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidInput, field)
	}
	return nil
}

func (u User) Validate() error {
	if err := ValidateText(u.Name, "name", true); err != nil {
		return err
	}
	if err := ValidateText(u.Username, "username", true); err != nil {
		return err
	}
	if len(u.Password) < 8 || len(u.Password) > 72 {
		return fmt.Errorf("%w: password must contain 8–72 bytes", ErrInvalidInput)
	}
	return nil
}

func (l TodoList) Validate() error {
	if err := ValidateText(l.Title, "title", true); err != nil {
		return err
	}
	return ValidateText(l.Description, "description", false)
}

func (i TodoItem) Validate() error {
	if err := ValidateText(i.Title, "title", true); err != nil {
		return err
	}
	return ValidateText(i.Description, "description", false)
}
