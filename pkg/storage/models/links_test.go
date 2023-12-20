package models_test

import (
	"testing"

	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

func TestLinks(t *testing.T) {
	testCases := []models.Model{
		&models.ShortURL{},
		&models.ShortURL{ID: 31342, URL: "https://rotational.io", Title: "Rotational"},
	}

	test := makeModelsTest(models.LinksBucket, testCases)
	test(t)
}
