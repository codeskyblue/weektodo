//go:build windows

package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func installDesktopAsset(archive, exe string) error {
	if exe == "" {
		return errors.New("executable path is unknown")
	}
	if _, err := os.Stat(archive); err != nil {
		return err
	}
	extracted, err := extractWindowsExecutable(archive, filepath.Dir(exe))
	if err != nil {
		return err
	}
	defer os.Remove(extracted)
	if err := replaceExecutable(extracted, exe); err != nil {
		return fmt.Errorf("install Windows update: %w", err)
	}
	return nil
}
