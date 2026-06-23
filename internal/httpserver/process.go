package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/leshchenko/pdf-extract/internal/config"
	"github.com/leshchenko/pdf-extract/internal/pdf"
	"github.com/leshchenko/pdf-extract/internal/storage"
)

// Service wires config, HTTP fetch, and storage.
type Service struct {
	Cfg         *config.Config
	FetchClient *http.Client
	Store       *storage.Storage
	Log         *slog.Logger
}

func (s *Service) logFromRequest(r *http.Request, level slog.Level, msg string, attrs ...any) {
	if s == nil || s.Log == nil {
		return
	}
	attrs = append(attrs, "request_id", middleware.GetReqID(r.Context()))
	s.Log.Log(r.Context(), level, msg, attrs...)
}

func (s *Service) failProcess(w http.ResponseWriter, r *http.Request, status int, title, detail string, err error) {
	if s != nil && s.Log != nil {
		level := slog.LevelWarn
		if status >= 500 {
			level = slog.LevelError
		}
		attrs := []any{"status", status, "problem_title", title}
		if err != nil {
			attrs = append(attrs, "err", err.Error())
		}
		s.logFromRequest(r, level, "process_failed", attrs...)
	}
	writeProblem(w, status, title, detail)
}

func (s *Service) logProcessOK(r *http.Request, start time.Time, sourceType string, renderImage, cropMargins bool, textLen int, imageID string, pdfBytes int64, extra ...any) {
	if s == nil || s.Log == nil {
		return
	}
	attrs := []any{
		"source_type", sourceType,
		"render_image", renderImage,
		"crop_margins", cropMargins,
		"duration_ms", time.Since(start).Milliseconds(),
		"text_len", textLen,
		"pdf_bytes", pdfBytes,
	}
	if imageID != "" {
		attrs = append(attrs, "image_id", imageID)
	}
	attrs = append(attrs, extra...)
	s.logFromRequest(r, slog.LevelInfo, "process_ok", attrs...)
}

func (s *Service) absFileURL(id string) string {
	base := strings.TrimRight(s.Cfg.PublicBaseURL, "/")
	return fmt.Sprintf("%s/v1/files/%s", base, id)
}

func validatePDFHeader(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 5)
	if _, err := io.ReadFull(f, buf); err != nil {
		return fmt.Errorf("read pdf header: %w", err)
	}
	if string(buf) != "%PDF-" {
		return fmt.Errorf("file is not a valid PDF")
	}
	return nil
}

func (s *Service) runPipeline(ctx context.Context, pdfPath string, renderImage, cropMargins bool) (text string, imageID string, outPNG string, err error) {
	st, statErr := os.Stat(pdfPath)
	if statErr != nil {
		return "", "", "", statErr
	}
	if st.Size() == 0 {
		return "", "", "", fmt.Errorf("PDF file is empty")
	}
	if err := validatePDFHeader(pdfPath); err != nil {
		return "", "", "", err
	}
	enc, err := pdf.IsEncrypted(ctx, pdfPath)
	if err != nil {
		return "", "", "", err
	}
	if enc {
		return "", "", "", fmt.Errorf("PDF is encrypted or password-protected")
	}
	text, err = pdf.ExtractText(ctx, pdfPath)
	if err != nil {
		return "", "", "", err
	}
	if !renderImage {
		return text, "", "", nil
	}
	id := uuid.NewString()
	outPath := filepath.Join(s.Cfg.OutputDir, id+".png")
	if err := pdf.StitchToPNG(ctx, pdfPath, outPath, cropMargins, s.Cfg.RenderDPI); err != nil {
		return "", "", "", err
	}
	return text, id, outPath, nil
}

// HandleProcessJSON handles application/json POST /v1/process.
func (s *Service) HandleProcessJSON(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req ProcessJSONRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.failProcess(w, r, http.StatusBadRequest, "Invalid JSON", err.Error(), err)
		return
	}
	if !strings.EqualFold(strings.TrimSpace(req.Source.Type), "url") {
		s.failProcess(w, r, http.StatusBadRequest, "Invalid source", `source.type must be "url" for JSON requests`, nil)
		return
	}
	urlStr := strings.TrimSpace(req.Source.URL)
	if urlStr == "" {
		s.failProcess(w, r, http.StatusBadRequest, "Invalid source", "source.url is required", nil)
		return
	}
	renderImage, cropMargins := req.Options.resolved()
	if !renderImage {
		cropMargins = false
	}

	urlHost := ""
	if u, err := url.Parse(urlStr); err == nil {
		urlHost = u.Hostname()
	}

	id := uuid.NewString()
	pdfPath := filepath.Join(s.Cfg.UploadDir, id+".pdf")
	if err := DownloadPDF(s.FetchClient, urlStr, s.Cfg.MaxDownloadBytes, pdfPath); err != nil {
		_ = os.Remove(pdfPath)
		s.failProcess(w, r, http.StatusBadRequest, "Failed to fetch PDF", err.Error(), err)
		return
	}

	text, imgID, outPNG, err := s.runPipeline(r.Context(), pdfPath, renderImage, cropMargins)
	if err != nil {
		_ = os.Remove(pdfPath)
		if outPNG != "" {
			_ = os.Remove(outPNG)
		}
		s.failProcess(w, r, http.StatusBadRequest, "PDF processing failed", err.Error(), err)
		return
	}

	pdfBytes := int64(0)
	if st, err := os.Stat(pdfPath); err == nil {
		pdfBytes = st.Size()
	}

	if renderImage && imgID != "" && outPNG != "" {
		s.Store.ScheduleDelete(pdfPath, outPNG)
	} else {
		s.Store.ScheduleDelete(pdfPath)
	}

	var img *ImageRef
	if renderImage && imgID != "" {
		img = &ImageRef{ID: imgID, URL: s.absFileURL(imgID)}
	}
	s.logProcessOK(r, start, "url", renderImage, cropMargins, len(text), imgID, pdfBytes, "url_host", urlHost)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ProcessResponse{
		Status: "success",
		Text:   text,
		Image:  img,
	})
}

// HandleProcessMultipart handles multipart/form-data POST /v1/process.
func (s *Service) HandleProcessMultipart(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if err := r.ParseMultipartForm(s.Cfg.MaxUploadBytes); err != nil {
		s.failProcess(w, r, http.StatusBadRequest, "Invalid multipart form", err.Error(), err)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		s.failProcess(w, r, http.StatusBadRequest, "Missing file", `form field "file" with PDF is required`, err)
		return
	}
	defer file.Close()

	opts := Options{}
	if raw := strings.TrimSpace(r.FormValue("options")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &opts); err != nil {
			s.failProcess(w, r, http.StatusBadRequest, "Invalid options JSON", err.Error(), err)
			return
		}
	}
	renderImage, cropMargins := opts.resolved()
	if !renderImage {
		cropMargins = false
	}

	id := uuid.NewString()
	pdfPath := filepath.Join(s.Cfg.UploadDir, id+".pdf")
	out, err := os.Create(pdfPath)
	if err != nil {
		s.failProcess(w, r, http.StatusInternalServerError, "Storage error", err.Error(), err)
		return
	}
	lim := io.LimitReader(file, s.Cfg.MaxUploadBytes+1)
	n, err := io.Copy(out, lim)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(pdfPath)
		s.failProcess(w, r, http.StatusBadRequest, "Failed to save upload", err.Error(), err)
		return
	}
	if n > s.Cfg.MaxUploadBytes {
		_ = os.Remove(pdfPath)
		s.failProcess(w, r, http.StatusRequestEntityTooLarge, "Upload too large", "file exceeds MAX_UPLOAD_BYTES", nil)
		return
	}
	if hdr.Size > 0 && hdr.Size > s.Cfg.MaxUploadBytes {
		_ = os.Remove(pdfPath)
		s.failProcess(w, r, http.StatusRequestEntityTooLarge, "Upload too large", "file exceeds MAX_UPLOAD_BYTES", nil)
		return
	}

	text, imgID, outPNG, err := s.runPipeline(r.Context(), pdfPath, renderImage, cropMargins)
	if err != nil {
		_ = os.Remove(pdfPath)
		if outPNG != "" {
			_ = os.Remove(outPNG)
		}
		s.failProcess(w, r, http.StatusBadRequest, "PDF processing failed", err.Error(), err)
		return
	}

	pdfBytes := int64(0)
	if st, err := os.Stat(pdfPath); err == nil {
		pdfBytes = st.Size()
	}

	if renderImage && imgID != "" && outPNG != "" {
		s.Store.ScheduleDelete(pdfPath, outPNG)
	} else {
		s.Store.ScheduleDelete(pdfPath)
	}

	var img *ImageRef
	if renderImage && imgID != "" {
		img = &ImageRef{ID: imgID, URL: s.absFileURL(imgID)}
	}
	filename := ""
	if hdr != nil {
		filename = hdr.Filename
	}
	s.logProcessOK(r, start, "upload", renderImage, cropMargins, len(text), imgID, pdfBytes, "filename", filename)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ProcessResponse{
		Status: "success",
		Text:   text,
		Image:  img,
	})
}
