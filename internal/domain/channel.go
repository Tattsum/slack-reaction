package domain

import "time"

// Channel はSlackチャンネルを表すドメインモデル
type Channel struct {
	ID   string
	Name string
}

// DateRange は日付範囲を表す値オブジェクト
type DateRange struct {
	Start time.Time
	End   time.Time
}

// IsValid は日付範囲が有効かどうかを検証する
func (dr *DateRange) IsValid() bool {
	return dr.Start.IsZero() || dr.End.IsZero() || dr.Start.Before(dr.End) || dr.Start.Equal(dr.End)
}

// Contains は指定された時刻が日付範囲内かどうかを返す
func (dr *DateRange) Contains(t time.Time) bool {
	if !dr.Start.IsZero() && t.Before(dr.Start) {
		return false
	}
	if !dr.End.IsZero() && t.After(dr.End) {
		return false
	}
	return true
}
