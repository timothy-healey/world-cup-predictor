package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordersUpdateGetters(t *testing.T) {
	Reset()
	defer Reset()

	require.Equal(t, -1, FDOrg().RemainingMinute)
	require.True(t, FDOrg().LastUpdated.IsZero())

	RecordFDOrg(4)
	require.Equal(t, 4, FDOrg().RemainingMinute)
	require.False(t, FDOrg().LastUpdated.IsZero())

	RecordOdds(120, 380, 1)
	o := Odds()
	require.Equal(t, 120, o.Remaining)
	require.Equal(t, 380, o.Used)
	require.Equal(t, 1, o.LastCost)
	require.False(t, o.LastUpdated.IsZero())
}

func TestResetClearsState(t *testing.T) {
	RecordFDOrg(7)
	RecordOdds(10, 490, 1)
	Reset()
	require.Equal(t, -1, FDOrg().RemainingMinute)
	require.Equal(t, -1, Odds().Remaining)
	require.True(t, FDOrg().LastUpdated.IsZero())
	require.True(t, Odds().LastUpdated.IsZero())
}
