package migrations_test

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	t.Run("MIG0000", func(t *testing.T) {
		db := makeFixtureDB(t, "testdata/mig0.tgz")

		// Check the fixture
		records, err := counts(db)
		require.NoError(t, err, "could not count contents of database")
		require.Len(t, records, 2, "unexpected number of objects, have the fixtures changed?")
		require.Equal(t, 6, records["links"], "unexpected number of links, have the fixtures changed?")
		require.Equal(t, 1, records["apikeys"], "unexpected number of apikeys, have the fixtures changed?")

		err = db.Update(migrations.Migrate)
		require.NoError(t, err)
	})
}

// Counts returns the counts in the database by key length
func counts(db *badger.DB) (map[string]int, error) {
	counter := make(map[string]int)
	err := db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()

			switch len(key) {
			case 16:
				counter["apikeys"]++
			case 8:
				counter["links"]++
			default:
				counter["unknown"]++
			}
		}
		return nil
	})

	return counter, err
}

// Create a badger database using the fixtures at a temporary directory.
func makeFixtureDB(t *testing.T, fixturePath string) (db *badger.DB) {
	dbpath := unzipFixtures(t, fixturePath)

	opts := badger.DefaultOptions(dbpath)
	opts.ReadOnly = false
	opts.Logger = nil

	db, err := badger.Open(opts)
	require.NoError(t, err, "could not open badger database")
	return db
}

// Unzip the fixture path to a temporary directory, ready to load a database.
func unzipFixtures(t *testing.T, fixturePath string) (dbpath string) {
	f, err := os.Open(fixturePath)
	require.NoError(t, err, "could not open fixture path")
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "could not open gzip stream")
	defer gz.Close()

	tarball := tar.NewReader(gz)
	dbpath = t.TempDir()

	for {
		header, err := tarball.Next()
		if err == io.EOF {
			break
		}

		require.NoError(t, err, "could not read next item in tarball")

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.Mkdir(filepath.Join(dbpath, header.Name), 0755)
			require.NoError(t, err, "could not make directory %s", header.Name)
		case tar.TypeReg:
			out, err := os.Create(filepath.Join(dbpath, header.Name))
			require.NoError(t, err, "could not create file %s", header.Name)

			_, err = io.Copy(out, tarball)
			require.NoError(t, err, "could not write contents to %s", header.Name)
			out.Close()
		default:
			require.NoError(t, fmt.Errorf("unknown type %d in %s", header.Typeflag, header.Name))
		}
	}

	return dbpath
}
