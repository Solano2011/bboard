package task

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"bboard/internal/domain/task"
	"bboard/internal/domain/user"

	"github.com/sirupsen/logrus"
)

const maxImageDimension = 10000

type UploadedPhotoFile struct {
	Data []byte
}

type UploadTaskPhotosInput struct {
	TaskID    int64
	ActorID   int64
	ActorRole user.Role
	Files     []UploadedPhotoFile
}

type UploadTaskPhotosUseCase struct {
	taskRepo     task.Repository
	photoStorage PhotoStorage
	log          *logrus.Entry
}

func NewUploadTaskPhotosUseCase(
	taskRepo task.Repository,
	photoStorage PhotoStorage,
	log *logrus.Logger,
) *UploadTaskPhotosUseCase {
	return &UploadTaskPhotosUseCase{
		taskRepo:     taskRepo,
		photoStorage: photoStorage,
		log:          log.WithField("usecase", "task.upload_photos"),
	}
}

func (uc *UploadTaskPhotosUseCase) Execute(ctx context.Context, input UploadTaskPhotosInput) ([]*task.Photo, error) {
	if input.TaskID <= 0 {
		return nil, task.ErrTaskNotFound
	}
	if len(input.Files) == 0 {
		return nil, task.ErrEmptyPhotoFile
	}
	if len(input.Files) > task.MaxPhotosPerTask {
		return nil, task.ErrTooManyPhotos
	}

	type preparedPhoto struct {
		mime string
		ext  string
		data []byte
	}
	prepared := make([]preparedPhoto, 0, len(input.Files))

	for _, f := range input.Files {
		if len(f.Data) == 0 {
			return nil, task.ErrEmptyPhotoFile
		}
		if len(f.Data) > task.MaxPhotoSizeBytes {
			return nil, task.ErrPhotoTooLarge
		}

		mimeType, ext, ok := detectAndValidateImage(f.Data)
		if !ok || !task.IsAllowedPhotoMimeType(mimeType) {
			return nil, task.ErrUnsupportedPhotoType
		}

		prepared = append(prepared, preparedPhoto{
			mime: mimeType,
			ext:  ext,
			data: f.Data,
		})
	}

	// Быстрая предварительная проверка прав/статуса до записи на диск
	t, err := uc.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	if err := t.CanManagePhotos(input.ActorID, input.ActorRole); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	photos := make([]*task.Photo, 0, len(prepared))
	savedKeys := make([]string, 0, len(prepared))

	cleanupFiles := func() {
		for _, key := range savedKeys {
			if delErr := uc.photoStorage.Delete(ctx, key); delErr != nil {
				uc.log.WithError(delErr).WithField("storage_key", key).Warn("failed to rollback saved photo file")
			}
		}
	}

	for _, p := range prepared {
		key, err := uc.photoStorage.Save(ctx, input.TaskID, p.ext, p.data)
		if err != nil {
			cleanupFiles()
			return nil, err
		}
		savedKeys = append(savedKeys, key)
		photos = append(photos, &task.Photo{
			TaskID:     input.TaskID,
			StorageKey: key,
			MimeType:   p.mime,
			SizeBytes:  int64(len(p.data)),
			CreatedAt:  now,
		})
	}

	if err := uc.taskRepo.AddPhotos(ctx, input.TaskID, input.ActorID, input.ActorRole, photos); err != nil {
		cleanupFiles()
		return nil, err
	}

	return uc.taskRepo.ListPhotosByTaskID(ctx, input.TaskID)
}

func detectAndValidateImage(data []byte) (mimeType string, ext string, ok bool) {
	detected := http.DetectContentType(data)
	switch detected {
	case "image/jpeg":
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
			return "", "", false
		}
		return "image/jpeg", ".jpg", true

	case "image/png":
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
			return "", "", false
		}
		return "image/png", ".png", true

	case "image/webp":
		if !isValidWebP(data) {
			return "", "", false
		}
		return "image/webp", ".webp", true

	default:
		return "", "", false
	}
}

func isValidWebP(data []byte) bool {
	if len(data) < 16 {
		return false
	}
	if !bytes.Equal(data[0:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
		return false
	}
	chunk := data[12:16]
	return bytes.Equal(chunk, []byte("VP8 ")) ||
		bytes.Equal(chunk, []byte("VP8L")) ||
		bytes.Equal(chunk, []byte("VP8X"))
}
