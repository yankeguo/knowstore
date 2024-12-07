package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResultSet(t *testing.T) {
	rs := NewResultSet()
	require.Zero(t, rs.Len())

	nn1 := NamespacedName{"ns", "n1"}
	nn2 := NamespacedName{"ns", "n2"}
	nn3 := NamespacedName{"ns", "n3"}

	rs.AddCID("c1", nn1)
	rs.AddCID("c2", nn1)
	rs.AddCID("c3", nn2)
	rs.AddCID("c4", nn2)
	rs.AddCID("c5", nn3)
	rs.AddCID("c6", nn3)

	require.Equal(t, 3, rs.Len())
	require.True(t, rs.HasCID("c1"))
	require.False(t, rs.HasCID("c7"))

	item, ok := rs.SaveUsage("c1", 1)
	require.Equal(t, nn1, item)
	require.True(t, ok)

	rs.SaveUsage("c2", 2)
	rs.SaveUsage("c3", 2)

	item, ok = rs.SaveUsage("c7", 2)
	require.False(t, ok)

	size, ok := rs.GetUsage(NamespacedName{"ns", "n4"})
	require.Zero(t, size)
	require.False(t, ok)

	size, ok = rs.GetUsage(nn1)
	require.Equal(t, int64(3), size)
	require.True(t, ok)

	size, ok = rs.GetUsage(nn2)
	require.Equal(t, int64(2), size)
	require.False(t, ok)

	rs.SaveUsage("c4", 3)
	size, ok = rs.GetUsage(nn2)
	require.Equal(t, int64(5), size)
	require.True(t, ok)
}
