package dao

func normalizePagination(page, pageSize, defaultPageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	offset := (page - 1) * pageSize
	return page, pageSize, offset
}
