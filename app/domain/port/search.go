package port

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	maxKnowledgeFilterTextLength = 128
	maxKnowledgeFilterTags       = 16
)

// KnowledgeFilter contains only user-facing filter values. It deliberately
// does not expose Meilisearch's filter DSL to the application layer.
type KnowledgeFilter struct {
	Category string
	Tags     []string
	Year     int
	From     time.Time
	To       time.Time
}

// KnowledgeFilterInput is the HTTP-friendly representation used by the
// application service. From is inclusive and To is exclusive; a date-only To
// value is converted to the next day so the whole requested date is included.
type KnowledgeFilterInput struct {
	Category string
	Tags     []string
	Year     string
	From     string
	To       string
}

func (f KnowledgeFilter) Empty() bool {
	return strings.TrimSpace(f.Category) == "" && len(f.Tags) == 0 && f.Year == 0 && f.From.IsZero() && f.To.IsZero()
}

// ParseKnowledgeFilter parses the public search parameters and normalizes
// them before they can reach an infrastructure adapter.
func ParseKnowledgeFilter(input KnowledgeFilterInput) (KnowledgeFilter, error) {
	filter := KnowledgeFilter{Category: input.Category}
	for _, raw := range input.Tags {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, value := range strings.Split(raw, ",") {
			value = strings.TrimSpace(value)
			if value == "" {
				return KnowledgeFilter{}, errors.New("tag filter contains an empty value")
			}
			filter.Tags = append(filter.Tags, value)
		}
	}
	if year := strings.TrimSpace(input.Year); year != "" {
		parsed, err := strconv.Atoi(year)
		if err != nil {
			return KnowledgeFilter{}, errors.New("year filter must be an integer")
		}
		if parsed < 1 || parsed > 9999 {
			return KnowledgeFilter{}, errors.New("year filter is out of range")
		}
		filter.Year = parsed
	}
	from, _, err := parseKnowledgeFilterTime(input.From)
	if err != nil {
		return KnowledgeFilter{}, err
	}
	to, toDateOnly, err := parseKnowledgeFilterTime(input.To)
	if err != nil {
		return KnowledgeFilter{}, err
	}
	if toDateOnly {
		to = to.AddDate(0, 0, 1)
	}
	filter.From = from
	filter.To = to
	return filter.Normalize()
}

// Normalize validates values supplied directly by non-HTTP callers and
// canonicalizes them for deterministic filter construction.
func (f KnowledgeFilter) Normalize() (KnowledgeFilter, error) {
	f.Category = strings.TrimSpace(f.Category)
	if err := validateKnowledgeFilterText(f.Category, "category"); err != nil {
		return KnowledgeFilter{}, err
	}
	if f.Year < 0 || f.Year > 9999 {
		return KnowledgeFilter{}, errors.New("year filter is out of range")
	}
	if len(f.Tags) > maxKnowledgeFilterTags {
		return KnowledgeFilter{}, errors.New("too many tag filters")
	}
	tags := make([]string, 0, len(f.Tags))
	seen := make(map[string]struct{}, len(f.Tags))
	for _, tag := range f.Tags {
		tag = strings.TrimSpace(tag)
		if err := validateKnowledgeFilterText(tag, "tag"); err != nil {
			return KnowledgeFilter{}, err
		}
		if tag == "" {
			return KnowledgeFilter{}, errors.New("tag filter must not be empty")
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	f.Tags = tags
	if !f.From.IsZero() {
		f.From = f.From.UTC()
	}
	if !f.To.IsZero() {
		f.To = f.To.UTC()
	}
	if !f.From.IsZero() && !f.To.IsZero() && !f.From.Before(f.To) {
		return KnowledgeFilter{}, errors.New("filter start time must be before end time")
	}
	return f, nil
}

func validateKnowledgeFilterText(value, name string) error {
	if value == "" {
		return nil
	}
	if len([]rune(value)) > maxKnowledgeFilterTextLength {
		return errors.New(name + " filter is too long")
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return errors.New(name + " filter contains a control character")
	}
	return nil
}

func parseKnowledgeFilterTime(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, false, nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed, true, nil
	}
	return time.Time{}, false, errors.New("time filter must use RFC3339 or YYYY-MM-DD")
}

// NormalizeSearchMode applies the public API default and rejects unknown
// modes before any infrastructure request is made.
func NormalizeSearchMode(value string) (SearchMode, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return SearchModeKeyword, nil
	}
	mode := SearchMode(value)
	switch mode {
	case SearchModeKeyword, SearchModeHybrid, SearchModeSemantic:
		return mode, nil
	default:
		return "", errors.New("search mode is invalid")
	}
}

// NormalizeKnowledgeQuery fills safe defaults for a provider-neutral search
// request. The concrete adapter still owns index capability checks and the
// mandatory public-article filter.
func (q KnowledgeQuery) Normalize(defaultIndex string) (KnowledgeQuery, error) {
	return q.NormalizeWithDefault(defaultIndex, DefaultSemanticRatio)
}

// NormalizeWithDefault applies a caller-selected hybrid ratio while keeping
// an omitted query ratio distinct from invalid values. A zero query ratio is
// the omission sentinel; explicit provider configuration must be positive and
// no greater than one.
func (q KnowledgeQuery) NormalizeWithDefault(defaultIndex string, defaultSemanticRatio float32) (KnowledgeQuery, error) {
	if defaultSemanticRatio <= 0 || defaultSemanticRatio > 1 {
		return KnowledgeQuery{}, errors.New("default semantic ratio must be between 0 and 1")
	}
	q.Index = strings.TrimSpace(q.Index)
	if q.Index == "" {
		q.Index = strings.TrimSpace(defaultIndex)
	}
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		return KnowledgeQuery{}, errors.New("search query is required")
	}
	mode, err := NormalizeSearchMode(string(q.Mode))
	if err != nil {
		return KnowledgeQuery{}, err
	}
	q.Mode = mode
	if q.Limit == 0 {
		q.Limit = DefaultKnowledgeSearchLimit
	}
	if q.Limit < 1 || q.Limit > MaxKnowledgeSearchLimit {
		return KnowledgeQuery{}, errors.New("search limit is out of range")
	}
	if q.SemanticRatio < 0 || q.SemanticRatio > 1 {
		return KnowledgeQuery{}, errors.New("semantic ratio must be between 0 and 1")
	}
	filter, err := q.Filter.Normalize()
	if err != nil {
		return KnowledgeQuery{}, err
	}
	q.Filter = filter
	switch q.Mode {
	case SearchModeKeyword:
		q.SemanticRatio = 0
	case SearchModeHybrid:
		if q.SemanticRatio == 0 {
			q.SemanticRatio = defaultSemanticRatio
		}
	case SearchModeSemantic:
		q.SemanticRatio = 1
	}
	return q, nil
}
