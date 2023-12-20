package models_test

import (
	"bytes"
	"testing"

	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/stretchr/testify/require"
)

func makeModelsTest(bucket models.Bucket, testCases []models.Model) func(t *testing.T) {
	return func(t *testing.T) {
		for i, model := range testCases {
			key := model.Key()
			require.Greater(t, len(key), 8, "test case %d does not have minimum key length of 8", i)
			require.True(t, bytes.HasPrefix(key, bucket[:]), "test case %d does not have correct prefix", i)

			data, err := model.MarshalValue()
			require.NoError(t, err, "could not marshal model in test case %d", i)
			require.NotZero(t, data, "no data returned from marshal in test case %d", i)

			var cmp models.Model
			switch model.(type) {
			case *models.ShortURL:
				cmp = &models.ShortURL{}
			case *models.APIKey:
				cmp = &models.APIKey{}
			default:
				require.Failf(t, "unknown model type", "test case %d had unknown type of model %T", i, model)
			}

			err = cmp.UnmarshalValue(data)
			require.NoError(t, err, "could not unmarshal model on test case %d", i)
			require.Equal(t, model, cmp, "unmarshal did not produce equal value to original in test case %d", i)
		}
	}
}
