package main

import (
	"fmt"
)

type BankAccount struct {
    Iban string
}

func NewBankAccount(iban string) (*BankAccount, error) {
    var validators ValidationSet
    validators.Add(MinLengthValidator{Min: 10})
    validators.Add(MaxLengthValidator{Max: 34})

    if err := validators.Validate(iban); err != nil {
        return nil, fmt.Errorf("iban validation failed: %w", err)
    }

    return &BankAccount{Iban: iban}, nil
}