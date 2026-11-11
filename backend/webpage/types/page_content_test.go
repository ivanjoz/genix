package types

import "testing"

// The packed ID is the table's only key: two (page, section) pairs must never share one.
func TestPageSectionIDIsUniquePerPageAndPosition(t *testing.T) {
	if got, want := PageSectionID(10, 1), int32(10001); got != want {
		t.Errorf("PageSectionID(10, 1) = %d, want %d", got, want)
	}
	if PageSectionID(10, MaxPageSections) >= PageSectionID(11, 1) {
		t.Error("the last section of a page must sort before the first of the next page")
	}
	if PageSectionID(32767, MaxPageSections) <= 0 {
		t.Error("the highest page and section overflowed int32")
	}
}
