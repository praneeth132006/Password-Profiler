package mutate

import "fmt"

// dobYearRange is how many years on either side of the target's birth year are
// derived as affixes ("±range"). Kept small to bound expansion.
const dobYearRange = 1

// YearAffixes builds a deterministic, de-duplicated list of year affixes in
// both 4-digit and 2-digit forms:
//
//   - if hasDOB: birthYear-range .. birthYear+range
//   - the current year (passed in for testability)
//   - each explicit extra year
//
// Example (birthYear 1990, currentYear 2025, extra [2024]):
//
//	1990 90 1989 89 1991 91 2025 25 2024 24
func YearAffixes(birthYear int, hasDOB bool, currentYear int, extra []int) []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(year int) {
		if year <= 0 {
			return
		}
		for _, form := range []string{
			fmt.Sprintf("%d", year),
			fmt.Sprintf("%02d", ((year%100)+100)%100),
		} {
			if _, ok := seen[form]; ok {
				continue
			}
			seen[form] = struct{}{}
			out = append(out, form)
		}
	}

	if hasDOB {
		add(birthYear)
		for d := 1; d <= dobYearRange; d++ {
			add(birthYear - d)
			add(birthYear + d)
		}
	}
	add(currentYear)
	for _, e := range extra {
		add(e)
	}
	return out
}
