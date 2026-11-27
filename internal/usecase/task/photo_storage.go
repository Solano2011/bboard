package task

import "context"

// PhotoStorage определяет контракт локального/облачного хранения файлов фотографий.
type PhotoStorage interface {
	Save(ctx context.Context, taskID int64, ext string, data []byte) (string, error)
	Delete(ctx context.Context, storageKey string) error
	DeleteTaskDir(ctx context.Context, taskID int64) error
}
