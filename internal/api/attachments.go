package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gogi0001/family-tasks/internal/events"
	"github.com/gogi0001/family-tasks/internal/models"
	"github.com/gogi0001/family-tasks/internal/storage"
)

// maxFormMemory — сколько байт multipart-формы держать в памяти.
// Всё, что больше, Go сохраняет во временный файл на диске.
// Это НЕ лимит размера файла; лимит задаётся через MaxBytesReader.
const maxFormMemory = 5 << 20 // 5 MB

var (
	errUnsupportedMime = errors.New("unsupported mime")
	allowedMimes       = map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}
)

type attachmentsHandler struct {
	tasks    storage.TaskStore
	atts     storage.AttachmentStore
	files    *storage.FileStorage
	events   *events.Hub
	maxBytes int64 // лимит размера одного файла
}

func attachmentURL(taskID, attID string) string {
	return fmt.Sprintf("/api/v1/tasks/%s/attachments/%s", taskID, attID)
}

func (h *attachmentsHandler) upload(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}

	taskID := r.PathValue("id")
	if _, err := h.tasks.Get(r.Context(), u.FamilyID, taskID); err != nil {
		writeStoreError(w, r, err)
		return
	}

	// MaxBytesReader возвращает 413 Payload Too Large при превышении.
	// Небольшой запас (+1 МБ) — на multipart-обвязку (boundary, заголовки полей).
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+(1<<20))

	if err := r.ParseMultipartForm(maxFormMemory); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("file too large (max %d MB)", h.maxBytes>>20))
			return
		}
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	att, err := h.saveOne(r.Context(), u, taskID, file, header)
	if err != nil {
		if errors.Is(err, errUnsupportedMime) {
			writeError(w, http.StatusBadRequest, "unsupported image type")
			return
		}
		slog.ErrorContext(r.Context(), "attachment upload", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.events.Broadcast(u.FamilyID, events.Event{Type: "task.updated"})
	writeJSON(w, http.StatusCreated, att)
}

func (h *attachmentsHandler) saveOne(
	ctx context.Context,
	u *models.User,
	taskID string,
	file multipart.File,
	header *multipart.FileHeader,
) (*models.Attachment, error) {
	// sniff mime из первых 512 байт
	buf := make([]byte, 512)
	n, err := io.ReadFull(file, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]

	mime := strings.Split(http.DetectContentType(buf), ";")[0]
	ext, ok := allowedMimes[mime]
	if !ok {
		return nil, errUnsupportedMime
	}

	reader := io.MultiReader(bytes.NewReader(buf), file)

	attID := uuid.NewString()
	relPath := attID + ext

	size, err := h.files.Save(relPath, reader)
	if err != nil {
		return nil, err
	}

	att := &models.Attachment{
		ID:         attID,
		TaskID:     taskID,
		Filename:   header.Filename,
		Mime:       mime,
		Size:       size,
		CreatedBy:  u.Name,
		CreatedAt:  time.Now().UTC(),
		StoredName: relPath,
		URL:        attachmentURL(taskID, attID),
	}

	if err := h.atts.Create(ctx, att); err != nil {
		_ = h.files.Remove(relPath)
		return nil, err
	}
	return att, nil
}

func (h *attachmentsHandler) serve(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	taskID := r.PathValue("id")
	attID := r.PathValue("attID")

	if _, err := h.tasks.Get(r.Context(), u.FamilyID, taskID); err != nil {
		writeStoreError(w, r, err)
		return
	}

	att, err := h.atts.Get(r.Context(), attID)
	if err != nil || att.TaskID != taskID {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	file, err := h.files.Open(att.StoredName)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "attachment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", att.Mime)
	w.Header().Set("Cache-Control", "private, max-age=31536000")
	http.ServeContent(w, r, att.Filename, att.CreatedAt, file)
}

func (h *attachmentsHandler) delete(w http.ResponseWriter, r *http.Request) {
	u, ok := requireFamily(w, r)
	if !ok {
		return
	}
	taskID := r.PathValue("id")
	attID := r.PathValue("attID")

	if _, err := h.tasks.Get(r.Context(), u.FamilyID, taskID); err != nil {
		writeStoreError(w, r, err)
		return
	}

	att, err := h.atts.Get(r.Context(), attID)
	if err != nil || att.TaskID != taskID {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	if err := h.atts.Delete(r.Context(), attID); err != nil {
		writeStoreError(w, r, err)
		return
	}
	_ = h.files.Remove(att.StoredName)

	h.events.Broadcast(u.FamilyID, events.Event{Type: "task.updated"})
	w.WriteHeader(http.StatusNoContent)
}
