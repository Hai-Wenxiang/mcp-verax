package assert

import "slices"

func contains(haystack []string, needle string) bool {
	return slices.Contains(haystack, needle)
}

func sortStrings(s []string) { slices.Sort(s) }

func sortedCopy(in []string) []string {
	out := slices.Clone(in)
	slices.Sort(out)
	return out
}

func equalSlices(a, b []string) bool {
	return slices.Equal(a, b)
}
