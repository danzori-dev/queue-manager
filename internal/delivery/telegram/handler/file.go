package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/mymmrac/telego"
)

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
