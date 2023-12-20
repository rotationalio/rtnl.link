package migrations

import (
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	// NOTE: Register migrations here in the order that they should be applied!
}

var (
	migrationKey = []byte{0x00, 0x00, 0x00, 0x00}
	migrations   = Migrations{}
)

func Register(migrate MigrateFn) {
	migration := &Migration{Migrate: migrate}
	if len(migrations) > 0 {
		migration.Previous = migrations[len(migrations)-1]
		migration.Version = migration.Previous.Version + 1
	} else {
		migration.Version = 1
		migration.Previous = nil
	}
	migrations = append(migrations, migration)
}

// Migration represents any changes that must be applied to the database using the
// MigrateFn defined in the migration. This data structure is saved in the datbase to
// identify what the database migration is currently at. To define a new migration,
// create a file with m000n.go then create a migration instance and register it. The
// store will automatically check the migration is at the latest version or apply the
// migrations in order if the database is not at the latest version.
type Migration struct {
	Version  uint16     `msgpack:"version"`
	Applied  time.Time  `msgpack:"applied"`
	Previous *Migration `msgpack:"previous"`
	Migrate  MigrateFn  `msgpack:"-"`
}

type Migrations []*Migration

type MigrateFn func(txn *badger.Txn) error

func (m *Migration) Key() []byte {
	return migrationKey
}

func (m *Migration) MarshalValue() ([]byte, error) {
	return msgpack.Marshal(m)
}

func (m *Migration) UnmarshalValue(data []byte) error {
	return msgpack.Unmarshal(data, m)
}

func (m Migrations) HasNext(n *Migration) bool {
	if n == nil || n.Version == 0 {
		return len(m) > 0
	}
	return len(m) > int(n.Version)
}

func (m Migrations) Next(n *Migration) *Migration {
	if n == nil || n.Version == 0 {
		if len(m) > 0 {
			return m[0]
		}
		return nil
	}

	// Because version is 1-indexed and the array is 0-indexed the "next" migration is
	// actually at the index of the current version. E.g. the index of n is version-1
	// and the next index is n.index+1 so version-1+1 == version.
	if len(m) > int(n.Version) {
		return m[n.Version]
	}
	return nil
}
