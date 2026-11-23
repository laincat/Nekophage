package collections

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxSnapshotBytes = 32 << 20

func readSnapshot(root string) ([]byte, error) {
	file, err := os.Open(filepath.Join(root, "collections.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSnapshotBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSnapshotBytes {
		return nil, errors.New("collections: snapshot exceeds 32 MiB")
	}
	return data, nil
}

func saveSnapshot(root string, data []byte) error {
	if len(data) > maxSnapshotBytes {
		return errors.New("collections: snapshot exceeds 32 MiB")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(root, ".collections-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), filepath.Join(root, "collections.json"))
}
