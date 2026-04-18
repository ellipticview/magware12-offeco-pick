package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenFileSourceLoad(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		dir := t.TempDir()
		filename := filepath.Join(dir, "token.txt")
		require.NoError(t, os.WriteFile(filename, []byte("  abc123 \n"), 0o600))

		source := NewTokenFileSource(filename)
		token, err := source.Load()

		require.NoError(t, err)
		require.Equal(t, "abc123", token)
	})

	t.Run("missing token file", func(t *testing.T) {
		source := NewTokenFileSource(filepath.Join(t.TempDir(), "missing-token.txt"))

		_, err := source.Load()

		require.Error(t, err)
		require.Contains(t, err.Error(), "kan")
	})

	t.Run("unreadable token path", func(t *testing.T) {
		source := NewTokenFileSource(t.TempDir())

		_, err := source.Load()

		require.Error(t, err)
		require.Contains(t, err.Error(), "kan")
	})

	t.Run("empty token file", func(t *testing.T) {
		dir := t.TempDir()
		filename := filepath.Join(dir, "token.txt")
		require.NoError(t, os.WriteFile(filename, []byte(" \n "), 0o600))

		source := NewTokenFileSource(filename)
		_, err := source.Load()

		require.Error(t, err)
		require.Contains(t, err.Error(), "leeg")
	})
}
