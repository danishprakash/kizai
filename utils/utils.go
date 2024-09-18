package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func XMLReadyString(data string) string {
	// Create a buffer to hold the escaped output
	var buf bytes.Buffer

	// Escape the text using xml.EscapeText
	err := xml.EscapeText(&buf, []byte(data))
	if err != nil {
		logrus.Errorf("Error escaping text:", err)
	}

	return buf.String()
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	// Copy modes.
	f, err := os.Stat(src)
	if err == nil {
		err = os.Chmod(dst, f.Mode())
		if err != nil {
			return err
		}
	}

	return out.Close()
}

func CopyDir(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("error: %q is not a directory", fi)
	}

	if err = os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	items, _ := os.ReadDir(src)
	for _, item := range items {
		srcFilename := filepath.Join(src, item.Name())
		dstFilename := filepath.Join(dst, item.Name())
		if item.IsDir() {
			if err := CopyDir(srcFilename, dstFilename); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcFilename, dstFilename); err != nil {
				return err
			}
		}
	}

	return nil
}
