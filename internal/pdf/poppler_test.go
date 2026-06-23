package pdf

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const testPDF = "../../testdata/onepage.pdf"

func TestExtractText_ContextTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	time.Sleep(2 * time.Microsecond) // ensure deadline passed
	_, err := ExtractText(ctx, testPDF)
	if err == nil {
		t.Fatal("expected error from expired context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context-related error, got: %v", err)
	}
}

func TestPageCount_ContextTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	time.Sleep(2 * time.Microsecond)
	_, err := PageCount(ctx, testPDF)
	if err == nil {
		t.Fatal("expected error from expired context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context-related error, got: %v", err)
	}
}

func TestIsEncrypted_ContextTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()
	time.Sleep(2 * time.Microsecond)
	_, err := IsEncrypted(ctx, testPDF)
	if err == nil {
		t.Fatal("expected error from expired context, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context-related error, got: %v", err)
	}
}

func TestExtractText_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	text, err := ExtractText(ctx, testPDF)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Minimal PDF has no real text, so empty is fine
	_ = text
}

func TestPageCount_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	n, err := PageCount(ctx, testPDF)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 page, got %d", n)
	}
}
