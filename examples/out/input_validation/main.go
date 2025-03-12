package main

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type PhoneNumber string

type BankAccountInput struct {
	Iban string
}

type CreateUserInput struct {
	ID           string
	Name         string
	PhoneNumbers []PhoneNumber
	BankAccount  BankAccountInput
}

type CreateUserOutput struct {
	Value float32
}

func validatePhoneNumber(phone PhoneNumber) error {
	str := string(phone)
	if len(str) < 3 || len(str) > 10 {
		return fmt.Errorf("phone number must be between 3 and 10 characters")
	}
	if !strings.HasPrefix(str, "+") && !strings.HasPrefix(str, "0") {
		return fmt.Errorf("phone number must start with + or 0")
	}
	return nil
}

func createUser(input CreateUserInput) (*CreateUserOutput, error) {
	// Validate ID (UUID v4)
	if _, err := uuid.Parse(input.ID); err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	// Validate name
	if len(input.Name) < 3 || len(input.Name) > 10 {
		return nil, fmt.Errorf("name must be between 3 and 10 characters")
	}

	// Validate phone numbers
	for _, phone := range input.PhoneNumbers {
		if err := validatePhoneNumber(phone); err != nil {
			return nil, fmt.Errorf("invalid phone number: %w", err)
		}
	}

	// Validate bank account IBAN
	if len(input.BankAccount.Iban) < 10 || len(input.BankAccount.Iban) > 34 {
		return nil, fmt.Errorf("IBAN must be between 10 and 34 characters")
	}

	fmt.Printf("Creating user with id: %s\n", input.ID)
	return &CreateUserOutput{Value: 300.3}, nil
}

func main() {
	input := CreateUserInput{
		ID:           "abcdef-abc1234-abc1234-abc1234",
		Name:         "Joe",
		PhoneNumbers: []PhoneNumber{"+491241234132"},
		BankAccount: BankAccountInput{
			Iban: "DE89370400440532013000",
		},
	}

	result, err := createUser(input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Result: %f\n", result.Value)
}