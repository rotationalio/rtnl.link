package migrations_test

import (
	"testing"

	. "github.com/rotationalio/rtnl.link/pkg/storage/migrations"
	"github.com/stretchr/testify/require"
)

type migrationTestCase struct {
	migration *Migration
	expected  *Migration
	hasNext   require.BoolAssertionFunc
}

func TestMigrations(t *testing.T) {
	makeTest := func(testCases []migrationTestCase, migrations Migrations) func(t *testing.T) {
		return func(t *testing.T) {
			for i, tc := range testCases {
				tc.hasNext(t, migrations.HasNext(tc.migration), "test case %d failed has next check", i)
				require.Equal(t, migrations.Next(tc.migration), tc.expected, "test case %d failed migration equality check", i)
			}
		}
	}

	emptyTestCases := []migrationTestCase{
		{nil, nil, require.False},
		{&Migration{}, nil, require.False},
		{&Migration{Version: 7}, nil, require.False},
	}
	t.Run("Empty", makeTest(emptyTestCases, Migrations{}))

	singleTestCases := []migrationTestCase{
		{nil, &Migration{Version: 1}, require.True},
		{&Migration{}, &Migration{Version: 1}, require.True},
		{&Migration{Version: 1}, nil, require.False},
		{&Migration{Version: 7}, nil, require.False},
	}
	t.Run("Single", makeTest(singleTestCases, Migrations{{Version: 1}}))

	multiMigrations := make(Migrations, 0, 5)
	for i := 0; i < 5; i++ {
		m := &Migration{Version: uint16(i + 1)}
		if i > 0 {
			m.Previous = multiMigrations[i-1]
		}
		multiMigrations = append(multiMigrations, m)
	}
	multiTestCases := []migrationTestCase{
		{nil, multiMigrations[0], require.True},
		{&Migration{}, multiMigrations[0], require.True},
		{&Migration{Version: 1}, multiMigrations[1], require.True},
		{&Migration{Version: 3}, multiMigrations[3], require.True},
		{&Migration{Version: uint16(len(multiMigrations)) - 1}, multiMigrations[len(multiMigrations)-1], require.True},
		{&Migration{Version: uint16(len(multiMigrations))}, nil, require.False},
		{&Migration{Version: uint16(len(multiMigrations)) + 1}, nil, require.False},
		{&Migration{Version: 7}, nil, require.False},
	}
	t.Run("Multiple", makeTest(multiTestCases, multiMigrations))
}
