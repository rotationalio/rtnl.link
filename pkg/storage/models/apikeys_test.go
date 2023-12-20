package models_test

import (
	"testing"
	"time"

	"github.com/rotationalio/rtnl.link/pkg/keygen"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/stretchr/testify/require"
)

func TestAssertAPIKeyLength(t *testing.T) {
	model := &models.APIKey{
		ClientID: keygen.KeyID(),
	}

	key := model.Key()
	require.Len(t, key, 20, "expected key to be 4+16 bytes")
}

func TestAPIKeys(t *testing.T) {
	testCases := []models.Model{
		&models.APIKey{ClientID: keygen.KeyID(), DerivedKey: keygen.Secret()},
		&models.APIKey{ClientID: keygen.KeyID(), DerivedKey: keygen.Secret(), Created: time.Now().Truncate(time.Millisecond), Modified: time.Now().Truncate(time.Millisecond)},
	}

	test := makeModelsTest(models.APIKeysBucket, testCases)
	test(t)
}
