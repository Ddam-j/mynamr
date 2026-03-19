//go:build !windows

package clip

import "errors"

type systemClipboard struct{}

func New() Interface {
	return systemClipboard{}
}

func (systemClipboard) ReadText() (string, error) {
	return "", errors.New("clipboard is not supported on this platform")
}

func (systemClipboard) WriteText(string) error {
	return errors.New("clipboard is not supported on this platform")
}
