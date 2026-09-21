package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStorage хранит загруженные файлы в каталоге root.
// Пути — относительные, вида "<uuid>.<ext>". Наружу из БД приходят только
// имена, которые мы сами и сгенерировали, так что path traversal невозможен.
type FileStorage struct {
	root string
}

func NewFileStorage(root string) (*FileStorage, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create uploads dir: %w", err)
	}
	return &FileStorage{root: root}, nil
}

// Save пишет содержимое r в <root>/relPath, возвращает размер.
func (f *FileStorage) Save(relPath string, r io.Reader) (int64, error) {
	abs := filepath.Join(f.root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return 0, err
	}
	file, err := os.Create(abs)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	n, err := io.Copy(file, r)
	if err != nil {
		_ = os.Remove(abs)
		return 0, err
	}
	return n, nil
}

// Open открывает файл для чтения (используется http.ServeContent).
func (f *FileStorage) Open(relPath string) (*os.File, error) {
	return os.Open(filepath.Join(f.root, relPath))
}

// Remove удаляет файл. Отсутствие файла — не ошибка.
func (f *FileStorage) Remove(relPath string) error {
	err := os.Remove(filepath.Join(f.root, relPath))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
