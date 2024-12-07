package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateDiskUsage(t *testing.T) {
	var size int64
	calculateDiskUsage(&size, filepath.Join("testdata", "diskusage"))
	require.Equal(t, int64(4), size)
}
