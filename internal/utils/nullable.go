package utils

import "database/sql"

// NullInt32 converts int to sql.NullInt32
// If v is 0, returns invalid NullInt32 (represents NULL)
func NullInt32(v int) sql.NullInt32 {
	if v == 0 {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(v), Valid: true}
}

// NullInt32Val converts sql.NullInt32 to int, returns 0 if not valid
func NullInt32Val(v sql.NullInt32) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int32)
}

// NullInt32Ptr converts sql.NullInt32 to *int, returns nil if not valid
func NullInt32Ptr(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}
	val := int(v.Int32)
	return &val
}
