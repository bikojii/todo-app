package todo

import (
	"errors"
	"strings"
	"testing"
)

func TestUpdateValidation(t *testing.T) {
	blank := " "
	long := strings.Repeat("я", 256)
	nul := "task\x00"
	valid := "Task"
	done := false
	for _, input := range []UpdateItemInput{{}, {Title: &blank}, {Description: &long}, {Title: &nul}} {
		if !errors.Is(input.Validate(), ErrInvalidInput) {
			t.Fatalf("accepted %+v", input)
		}
	}
	if err := (UpdateItemInput{Done: &done}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (UpdateListInput{Title: &valid}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (UpdateListInput{}).Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}
