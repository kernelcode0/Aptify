package deb

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestArReader(t *testing.T) {
	// Test files with odd and even lengths:
	files := []struct {
		name    string
		content string
	}{
		{"file1", "hello"},       // 5 bytes (odd)
		{"file2", "even"},        // 4 bytes (even)
		{"file3", "testing"},     // 7 bytes (odd)
		{"file4", "aligned1234"}, // 11 bytes (odd)
	}

	var arData bytes.Buffer
	arData.WriteString("!<arch>\n")
	for _, f := range files {
		hdr := fmt.Sprintf("%-16s%-12s%-6s%-6s%-8s%-10d`\n", f.name+"/", "0", "0", "0", "100644", len(f.content))
		arData.WriteString(hdr)
		arData.WriteString(f.content)
		if len(f.content)%2 != 0 {
			arData.WriteByte('\n')
		}
	}

	ar := newArReader(bytes.NewReader(arData.Bytes()))

	for _, expected := range files {
		name, rc, err := ar.Next()
		if err != nil {
			t.Fatalf("expected next file %s, got error: %v", expected.name, err)
		}
		if name != expected.name {
			t.Errorf("expected name %q, got %q", expected.name, name)
		}

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(data) != expected.content {
			t.Errorf("file %s: expected content %q, got %q", expected.name, expected.content, string(data))
		}
		if err := rc.Close(); err != nil {
			t.Errorf("close %s: %v", expected.name, err)
		}
	}

	// Next call should return io.EOF
	_, _, err := ar.Next()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestArReaderCloseDrain(t *testing.T) {
	files := []struct {
		name    string
		content string
	}{
		{"f1", "oddlengthcontent"}, // 17 bytes (odd)
		{"f2", "secondfile"},       // 10 bytes (even)
	}

	var arData bytes.Buffer
	arData.WriteString("!<arch>\n")
	for _, f := range files {
		hdr := fmt.Sprintf("%-16s%-12s%-6s%-6s%-8s%-10d`\n", f.name+"/", "0", "0", "0", "100644", len(f.content))
		arData.WriteString(hdr)
		arData.WriteString(f.content)
		if len(f.content)%2 != 0 {
			arData.WriteByte('\n')
		}
	}

	ar := newArReader(bytes.NewReader(arData.Bytes()))

	// Read only 2 bytes from f1, then Close() immediately
	name, rc, err := ar.Next()
	if err != nil {
		t.Fatalf("Next f1: %v", err)
	}
	if name != "f1" {
		t.Fatalf("got name %q, want f1", name)
	}
	buf := make([]byte, 2)
	if _, err := io.ReadFull(rc, buf); err != nil {
		t.Fatalf("read 2 bytes: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Should still cleanly read f2
	name2, rc2, err := ar.Next()
	if err != nil {
		t.Fatalf("Next f2: %v", err)
	}
	if name2 != "f2" {
		t.Fatalf("got name %q, want f2", name2)
	}
	all2, err := io.ReadAll(rc2)
	if err != nil {
		t.Fatalf("read f2: %v", err)
	}
	if string(all2) != "secondfile" {
		t.Errorf("got %q, want secondfile", string(all2))
	}
}

func TestArReaderInvalidSig(t *testing.T) {
	ar := newArReader(bytes.NewReader([]byte("not-ar-file-data")))
	_, _, err := ar.Next()
	if err == nil {
		t.Error("expected error for invalid archive signature, got nil")
	}
}
