package models_test

import (
	"testing"

	"github.com/rotationalio/rtnl.link/pkg/keygen"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/stretchr/testify/require"
)

func TestAssertAPIKeyLength(t *testing.T) {
	model := &models.APIKey{
		ClientID: keygen.KeyID(),
	}

	key, err := model.Key()
	require.NoError(t, err, "could not create key")
	require.Len(t, key, 16, "expected key to be 16 bytes")
}
