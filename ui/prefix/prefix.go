package prefix

import (
	"github.com/charmbracelet/lipgloss"
)

// MinPrefixLen is the minimum prefix length to show (for readability)
const MinPrefixLen = 1

// ComputeUniquePrefixes returns a map of ID -> minimum unique prefix length.
// For each ID, it finds the minimum number of characters needed to distinguish
// it from all other IDs in the set.
func ComputeUniquePrefixes(ids []string) map[string]int {
	result := make(map[string]int)

	for _, id := range ids {
		if id == "" {
			continue
		}
		minLen := MinPrefixLen
		for _, other := range ids {
			if other == "" || other == id {
				continue
			}
			// Find common prefix length
			commonLen := 0
			for i := 0; i < len(id) && i < len(other); i++ {
				if id[i] == other[i] {
					commonLen++
				} else {
					break
				}
			}
			// Need commonLen+1 to distinguish from this other ID
			needed := commonLen + 1
			if needed > minLen {
				minLen = needed
			}
		}
		// Cap at actual ID length
		if minLen > len(id) {
			minLen = len(id)
		}
		result[id] = minLen
	}

	return result
}

// FormatWithPrefix renders an ID with the unique prefix in one style and the rest in another.
func FormatWithPrefix(id string, prefixLen int, prefixStyle, restStyle lipgloss.Style) string {
	if id == "" {
		return ""
	}
	if prefixLen <= 0 {
		prefixLen = MinPrefixLen
	}
	if prefixLen >= len(id) {
		return prefixStyle.Render(id)
	}
	prefix := prefixStyle.Render(id[:prefixLen])
	rest := restStyle.Render(id[prefixLen:])
	return prefix + rest
}

// IDSet holds IDs and their computed unique prefix lengths for efficient lookup.
type IDSet struct {
	prefixes map[string]int
}

// NewIDSet creates a new IDSet from a slice of IDs.
func NewIDSet(ids []string) *IDSet {
	return &IDSet{
		prefixes: ComputeUniquePrefixes(ids),
	}
}

// PrefixLen returns the unique prefix length for the given ID.
// Returns MinPrefixLen if the ID is not in the set.
func (s *IDSet) PrefixLen(id string) int {
	if len, ok := s.prefixes[id]; ok {
		return len
	}
	return MinPrefixLen
}

// Format renders the ID with the unique prefix highlighted.
func (s *IDSet) Format(id string, prefixStyle, restStyle lipgloss.Style) string {
	return FormatWithPrefix(id, s.PrefixLen(id), prefixStyle, restStyle)
}
