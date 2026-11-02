package storage

import (
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Storage struct {
	imagesDir string
}

func NewStorage(imagesDir string) *Storage {
	return &Storage{imagesDir: imagesDir}
}

func (s *Storage) EnsureDirs() error {
	return os.MkdirAll(s.imagesDir, 0o755)
}

func (s *Storage) EnsureDirsFor(dirs ...string) error {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// SaveFiles guarda varios archivos en la carpeta por defecto de imágenes.
func (s *Storage) SaveFiles(files []*multipart.FileHeader, prefix string, allowedExts map[string]bool) ([]string, error) {
	return s.SaveFilesIn(s.imagesDir, files, prefix, allowedExts)
}

// SaveFilesIn guarda varios archivos en una carpeta concreta. Si alguno falla,
// elimina los ya escritos (rollback de esta operación) y devuelve el error.
// allowedExts restringe extensiones; si es nil acepta todo.
func (s *Storage) SaveFilesIn(dir string, files []*multipart.FileHeader, prefix string, allowedExts map[string]bool) ([]string, error) {
	var saved []string
	for i, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if allowedExts != nil && !allowedExts[ext] {
			s.removeSavedIn(dir, saved)
			return nil, fmt.Errorf("extensión %q no permitida", ext)
		}

		name := fmt.Sprintf("%s_%d_%d%s", prefix, time.Now().UnixMilli(), i, ext)
		if err := s.saveFileIn(dir, fh, name); err != nil {
			s.removeSavedIn(dir, saved)
			return nil, err
		}
		saved = append(saved, name)
	}
	return saved, nil
}

func (s *Storage) saveFileIn(dir string, fh *multipart.FileHeader, name string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(filepath.Join(dir, name))
		return err
	}
	return dst.Close()
}

func (s *Storage) removeSavedIn(dir string, saved []string) {
	for _, name := range saved {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// RemoveImages borra archivos de la carpeta por defecto de imágenes.
func (s *Storage) RemoveImages(filenames ...string) {
	s.RemoveIn(s.imagesDir, filenames...)
}

// RemoveIn borra archivos de una carpeta concreta (best-effort).
func (s *Storage) RemoveIn(dir string, filenames ...string) {
	for _, name := range filenames {
		if name == "" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			slog.Warn("no se pudo eliminar archivo", "dir", dir, "file", name, "error", err)
		}
	}
}

// MoveFile mueve un archivo entre carpetas. Si el origen no existe, no hace
// nada (se asume que ya está en el destino o no existe).
func (s *Storage) MoveFile(srcDir, dstDir, filename string) error {
	if filename == "" {
		return nil
	}

	src := filepath.Join(srcDir, filename)
	dst := filepath.Join(dstDir, filename)

	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		_ = os.Remove(dst)
	}
	return os.Rename(src, dst)
}

// MoveFileIfMissing mueve el archivo solo si no existe en el destino. Útil para
// reparaciones defensivas sin sobrescribir una copia válida.
func (s *Storage) MoveFileIfMissing(srcDir, dstDir, filename string) error {
	if filename == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dstDir, filename)); err == nil {
		return nil
	}
	return s.MoveFile(srcDir, dstDir, filename)
}

func (s *Storage) ImageURL(filename string) string {
	return "/imagenes/" + filename
}
