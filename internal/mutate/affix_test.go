package mutate

import (
	"reflect"
	"testing"
)

func TestYearAffixes(t *testing.T) {
	tests := []struct {
		name        string
		birthYear   int
		hasDOB      bool
		currentYear int
		extra       []int
		want        []string
	}{
		{
			name:        "dob range plus current plus extra",
			birthYear:   1990,
			hasDOB:      true,
			currentYear: 2025,
			extra:       []int{2024},
			// 1990/90, then -1: 1989/89, +1: 1991/91, current 2025/25, extra 2024/24
			want: []string{"1990", "90", "1989", "89", "1991", "91", "2025", "25", "2024", "24"},
		},
		{
			name:        "extras only, no dob, no current",
			hasDOB:      false,
			currentYear: 0,
			extra:       []int{2000},
			want:        []string{"2000", "00"},
		},
		{
			name:        "dedup across forms",
			birthYear:   2024,
			hasDOB:      true,
			currentYear: 2024, // same as birth -> deduped
			want:        []string{"2024", "24", "2023", "23", "2025", "25"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := YearAffixes(tt.birthYear, tt.hasDOB, tt.currentYear, tt.extra)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("YearAffixes = %v, want %v", got, tt.want)
			}
		})
	}
}
