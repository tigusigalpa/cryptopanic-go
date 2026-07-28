package cryptopanic

import "time"

// BoolPtr returns a pointer to the provided bool value.
// Useful for populating optional pointer fields in PostsQuery.
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to the provided int value.
func IntPtr(i int) *int {
	return &i
}

// TimePtr returns a pointer to the provided time.Time value.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// StringPtr returns a pointer to the provided string value.
func StringPtr(s string) *string {
	return &s
}
