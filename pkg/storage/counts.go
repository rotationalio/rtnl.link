package storage

import (
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

func (s *Store) Counts() (c *models.Counts, err error) {
	c = &models.Counts{}

	err = s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := models.LinksBucket[:]
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			obj := &models.ShortURL{}

			err := item.Value(func(v []byte) error {
				return obj.UnmarshalValue(v)
			})

			if err != nil {
				return err
			}

			if !obj.Expires.IsZero() && obj.Expires.Before(time.Now()) {
				// this is an expired link, so skip it in the counts
				continue
			}

			c.Links++
			c.Clicks += obj.Visits
			c.Campaigns += uint64(len(obj.Campaigns))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return c, nil
}
