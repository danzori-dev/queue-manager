package handler

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

const maxStudentsFileSize = 1 << 20

func (h *Handler) HandleStudents(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	return h.sendHTML(ctx, msg, texts.StudentList(h.students.List()))
}

func (h *Handler) HandleImportStudents(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if msg.Document == nil {
		return h.sendHTML(ctx, msg, texts.ImportStudentUsage)
	}

	raw, err := h.downloadFile(ctx, msg.Document.FileID, maxStudentsFileSize)
	if err != nil {
		return err
	}

	list, err := parseStudents(raw)
	if err != nil {
		return h.sendHTML(ctx, msg, texts.StudentsImportFailed(err))
	}

	if err := h.students.Import(ctx, msg.From.ID, list); err != nil {
		if errors.Is(err, access.ErrForbidden) {
			return h.sendHTML(ctx, msg, texts.AccessDenied)
		}

		return h.sendHTML(ctx, msg, texts.StudentsImportFailed(err))
	}

	return h.sendHTML(ctx, msg, texts.StudentsImported(len(list)))
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
