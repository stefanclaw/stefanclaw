package memory

import "strings"

// Search returns memory entries matching the keyword (case-insensitive).
func (s *Store) Search(keyword string) ([]string, error) {
	entries, err := s.Entries()
	if err != nil {
		return nil, err
	}

	keyword = strings.ToLower(keyword)
	var matches []string
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry), keyword) {
			matches = append(matches, entry)
		}
	}
	return matches, nil
}
