package pagination

const (
	// DefaultPageSize is used when page_size is absent or < 1.
	DefaultPageSize = 10
	// MaxPageSize caps page_size so clients can't request unbounded results.
	MaxPageSize = 100
)

// Normalize clamps page/pageSize to sane bounds: page >= 1 and
// 1 <= pageSize <= MaxPageSize.
func Normalize(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}
