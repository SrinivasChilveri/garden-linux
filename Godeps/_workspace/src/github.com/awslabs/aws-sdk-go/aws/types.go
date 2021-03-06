package aws

import (
	"fmt"
	"io"
	"time"
)

// String converts a Go string into a string pointer.
func String(v string) *string {
	return &v
}

// Boolean converts a Go bool into a boolean pointer.
func Boolean(v bool) *bool {
	return &v
}

// Long converts a Go int64 into a long pointer.
func Long(v int64) *int64 {
	return &v
}

// Double converts a Go float64 into a double pointer.
func Double(v float64) *float64 {
	return &v
}

// Time converts a Go Time into a Time pointer
func Time(t time.Time) *time.Time {
	return &t
}

func ReadSeekCloser(r io.Reader) ReaderSeekerCloser {
	return ReaderSeekerCloser{r}
}

type ReaderSeekerCloser struct {
	r io.Reader
}

func (r ReaderSeekerCloser) Read(p []byte) (int, error) {
	switch t := r.r.(type) {
	case io.Reader:
		return t.Read(p)
	}
	return 0, nil
}

func (r ReaderSeekerCloser) Seek(offset int64, whence int) (int64, error) {
	switch t := r.r.(type) {
	case io.Seeker:
		return t.Seek(offset, whence)
	}
	return int64(0), nil
}

func (r ReaderSeekerCloser) Close() error {
	switch t := r.r.(type) {
	case io.Closer:
		return t.Close()
	}
	return nil
}

// A SettableBool provides a boolean value which includes the state if
// the value was set or unset.  The set state is in addition to the value's
// value(true|false)
type SettableBool struct {
	value bool
	set   bool
}

// SetBool returns a SettableBool with a value set
func SetBool(value bool) SettableBool {
	return SettableBool{value: value, set: true}
}

// Get returns the value. Will always be false if the SettableBool was not set.
func (b *SettableBool) Get() bool {
	if !b.set {
		return false
	}
	return b.value
}

// Set sets the value and updates the state that the value has been set.
func (b *SettableBool) Set(value bool) {
	b.value = value
	b.set = true
}

// IsSet returns if the value has been set
func (b *SettableBool) IsSet() bool {
	return b.set
}

// Reset resets the state and value of the SettableBool to its initial default
// state of not set and zero value.
func (b *SettableBool) Reset() {
	b.value = false
	b.set = false
}

// String returns the string representation of the value if set. Zero if not set.
func (b *SettableBool) String() string {
	return fmt.Sprintf("%t", b.Get())
}

// GoString returns the string representation of the SettableBool value and state
func (b *SettableBool) GoString() string {
	return fmt.Sprintf("Bool{value:%t, set:%t}", b.value, b.set)
}
