//go:build !gohook

package input

import "errors"

var ErrOSInputUnavailable = errors.New("OS input support requires the gohook build tag")

// NewGlobalSource is unavailable in the default build.
func NewGlobalSource() (Source, error) {
	return nil, ErrOSInputUnavailable
}
