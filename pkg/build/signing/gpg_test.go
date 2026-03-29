package signing

import (
	"context"
	"testing"

	"dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestGPG_GPGSignerName_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	assert.Equal(t, "gpg", s.Name())
}

func TestGPG_GPGSignerAvailable_Good(t *testing.T) {
	s := NewGPGSigner("ABCD1234")
	_ = s.Available()
}

func TestGPG_GPGSignerNoKey_Bad(t *testing.T) {
	s := NewGPGSigner("")
	assert.False(t, s.Available())
}

func TestGPG_GPGSignerSign_Bad(t *testing.T) {
	fs := io.Local
	t.Run("fails when no key", func(t *testing.T) {
		s := NewGPGSigner("")
		err := s.Sign(context.Background(), fs, "test.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available or key not configured")
	})
}
