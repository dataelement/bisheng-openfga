package dm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openfga/openfga/pkg/storage/sqlcommon"
	"github.com/openfga/openfga/pkg/storage/test"
)

func TestDMDatastore(t *testing.T) {
	uri := os.Getenv("OPENFGA_DM_URI")
	if uri == "" {
		t.Skip("OPENFGA_DM_URI not set — skipping DaMeng integration tests")
	}

	cfg := sqlcommon.NewConfig()
	ds, err := New(uri, cfg)
	require.NoError(t, err)
	defer ds.Close()

	test.RunAllTests(t, ds)
}
