package student

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

const maxStudentsFileSize = 1 << 20

var errFileTooLarge = errors.New("file is too large")

type Deps struct {
	Replier    *reply.Replier
	Students   StudentService
	Bot        *telego.Bot
	APITimeout time.Duration
}

type Handler struct {
	reply      *reply.Replier
	students   StudentService
	bot        *telego.Bot
	apiTimeout time.Duration
}

func New(d Deps) *Handler {
	return &Handler{
		reply:      d.Replier,
		students:   d.Students,
		bot:        d.Bot,
		apiTimeout: d.APITimeout,
	}
}

func (h *Handler) List(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	return h.reply.SendHTML(ctx, msg, texts.StudentList(h.students.List()))
}

func (h *Handler) Import(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if msg.Document == nil {
		return h.reply.SendHTML(ctx, msg, texts.ImportStudentUsage)
	}

	raw, err := h.downloadFile(ctx, msg.Document.FileID, maxStudentsFileSize)
	if err != nil {
		return err
	}

	list, err := parseStudents(raw)
	if err != nil {
		return h.reply.SendHTML(ctx, msg, texts.StudentsImportFailed(err))
	}

	if err := h.students.Import(ctx, list); err != nil {
		return h.reply.SendHTML(ctx, msg, texts.StudentsImportFailed(err))
	}

	return h.reply.SendHTML(ctx, msg, texts.StudentsImported(len(list)))
}

func (h *Handler) downloadFile(ctx context.Context, fileID string, maxSize int64) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, h.apiTimeout)
	defer cancel()

	file, err := h.bot.GetFile(reqCtx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, h.bot.FileDownloadURL(file.FilePath), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download file: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if int64(len(data)) > maxSize {
		return nil, errFileTooLarge
	}

	return data, nil
}

func parseStudents(raw []byte) (map[int64]string, error) {
	var list map[int64]string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	for id, name := range list {
		list[id] = strings.TrimSpace(name)
	}

	return list, nil
}
