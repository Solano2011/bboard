package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LocalPhotoStorage сохраняет и удаляет фотографии объявлений в локальной директории сервера.
type LocalPhotoStorage struct {
	baseDir string
}

// NewLocalPhotoStorage создаёт локальное файловое хранилище в указанной директории.
func NewLocalPhotoStorage(baseDir string) (*LocalPhotoStorage, error) {
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return nil, fmt.Errorf("resolve upload base dir: %w", err)
	}
	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return nil, fmt.Errorf("create upload base dir: %w", err)
	}
	return &LocalPhotoStorage{baseDir: absBase}, nil
}

// Save записывает байты изображения в tasks/<taskID>/<random><ext> и возвращает относительный storage_key.
func (s *LocalPhotoStorage) Save(_ context.Context, taskID int64, ext string, data []byte) (string, error) {
	if taskID <= 0 {
		return "", fmt.Errorf("invalid task id for storage: %d", taskID)
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	rnd := make([]byte, 16)
	if _, err := rand.Read(rnd); err != nil {
		return "", fmt.Errorf("generate random photo filename: %w", err)
	}

	relKey := fmt.Sprintf("tasks/%d/%s%s", taskID, hex.EncodeToString(rnd), ext)
	fullPath, err := s.resolveSafePath(relKey)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("create task upload dir: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write photo file: %w", err)
	}

	return relKey, nil
}

// Delete удаляет файл по относительному storage_key с проверкой границ директории.
func (s *LocalPhotoStorage) Delete(_ context.Context, storageKey string) error {
	fullPath, err := s.resolveSafePath(storageKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove photo file: %w", err)
	}
	return nil
}

// DeleteTaskDir удаляет директорию uploads/tasks/<taskID> со всеми файлами.
func (s *LocalPhotoStorage) DeleteTaskDir(_ context.Context, taskID int64) error {
	if taskID <= 0 {
		return nil
	}
	relDir := filepath.Join("tasks", strconv.FormatInt(taskID, 10))
	fullDir, err := s.resolveSafePath(relDir)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(fullDir); err != nil {
		return fmt.Errorf("remove task upload dir: %w", err)
	}
	return nil
}

func (s *LocalPhotoStorage) resolveSafePath(storageKey string) (string, error) {
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(storageKey)))
	if cleanRel == "." || cleanRel == "" || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
		return "", fmt.Errorf("invalid storage key: %q", storageKey)
	}
	fullPath := filepath.Clean(filepath.Join(s.baseDir, cleanRel))
	relToBase, err := filepath.Rel(s.baseDir, fullPath)
	if err != nil || strings.HasPrefix(relToBase, "..") {
		return "", fmt.Errorf("path traversal detected for storage key: %q", storageKey)
	}
	return fullPath, nil
}
