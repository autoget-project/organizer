// Package ptr provides small pointer helpers shared across packages.
package ptr

// Str returns a pointer to the given string.
func Str(s string) *string { return &s }
