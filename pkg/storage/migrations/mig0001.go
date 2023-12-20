package migrations

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

// Migration0001 converts the key-length based identification of keyspace to a bucket
// oriented approach with prefixes that will allow us to list objects in the database.
func Migration0001(txn *badger.Txn) error {
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	defer iter.Close()

	// Record the number of items migrated
	counts := make(map[string]int)

	// Iterate over all keys in the database, rekeying them to their new prefix.
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		key := item.Key()

		var model models.Model
		switch len(key) {
		case 8:
			// This is a shorturl model based on the key length for v0
			model = &models.ShortURL{}
			counts["ShortURL"]++
		case 16:
			// This is an apikey model based on the key length for v0
			model = &models.APIKey{}
			counts["APIKey"]++
		default:
			return fmt.Errorf("unknown model key length %d", len(key))
		}

		if err := item.Value(model.UnmarshalValue); err != nil {
			return err
		}

		// Create the new item
		value, _ := model.MarshalValue()
		entry := badger.NewEntry(model.Key(), value)
		entry.WithMeta(item.UserMeta())

		if expires := item.ExpiresAt(); expires > 0 {
			now := uint64(time.Now().Unix())
			if expires > now {
				entry.WithTTL(time.Second * time.Duration(expires-now))
			}
		}

		if err := txn.SetEntry(entry); err != nil {
			return err
		}

		// Delete the old key
		if err := txn.Delete(key); err != nil {
			return err
		}
	}

	return nil
}
