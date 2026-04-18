package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type TokenSource interface {
	Load() (string, error)
}

type TokenFileSource struct {
	filename string
}

func NewTokenFileSource(filename string) *TokenFileSource {
	return &TokenFileSource{filename: filename}
}

func (s *TokenFileSource) Load() (string, error) {
	content, err := os.ReadFile(s.filename)
	if err != nil {
		return "", fmt.Errorf("kan %s niet lezen: %w", s.filename, err)
	}

	token := strings.TrimSpace(string(content))
	if token == "" {
		return "", errors.New("tokenbestand is leeg")
	}

	return token, nil
}
