package domain

import (
	"testing"
	"time"
)

func TestDateRange_IsValid(t *testing.T) {
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		name     string
		dateRange *DateRange
		expected bool
	}{
		{
			name: "有効な範囲",
			dateRange: &DateRange{
				Start: yesterday,
				End:   now,
			},
			expected: true,
		},
		{
			name: "開始日がゼロ",
			dateRange: &DateRange{
				Start: time.Time{},
				End:   now,
			},
			expected: true,
		},
		{
			name: "終了日がゼロ",
			dateRange: &DateRange{
				Start: now,
				End:   time.Time{},
			},
			expected: true,
		},
		{
			name: "開始日と終了日が同じ",
			dateRange: &DateRange{
				Start: now,
				End:   now,
			},
			expected: true,
		},
		{
			name: "開始日が終了日より後（無効）",
			dateRange: &DateRange{
				Start: tomorrow,
				End:   now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dateRange.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDateRange_Contains(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)
	dayAfterTomorrow := now.Add(48 * time.Hour)

	tests := []struct {
		name     string
		dateRange *DateRange
		time     time.Time
		expected bool
	}{
		{
			name: "範囲内",
			dateRange: &DateRange{
				Start: yesterday,
				End:   tomorrow,
			},
			time:     now,
			expected: true,
		},
		{
			name: "開始日より前",
			dateRange: &DateRange{
				Start: now,
				End:   tomorrow,
			},
			time:     yesterday,
			expected: false,
		},
		{
			name: "終了日より後",
			dateRange: &DateRange{
				Start: yesterday,
				End:   now,
			},
			time:     tomorrow,
			expected: false,
		},
		{
			name: "開始日と一致",
			dateRange: &DateRange{
				Start: now,
				End:   tomorrow,
			},
			time:     now,
			expected: true,
		},
		{
			name: "終了日と一致",
			dateRange: &DateRange{
				Start: yesterday,
				End:   now,
			},
			time:     now,
			expected: true,
		},
		{
			name: "開始日がゼロ（無制限）",
			dateRange: &DateRange{
				Start: time.Time{},
				End:   now,
			},
			time:     yesterday,
			expected: true,
		},
		{
			name: "終了日がゼロ（無制限）",
			dateRange: &DateRange{
				Start: now,
				End:   time.Time{},
			},
			time:     dayAfterTomorrow,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dateRange.Contains(tt.time); got != tt.expected {
				t.Errorf("Contains() = %v, want %v", got, tt.expected)
			}
		})
	}
}
