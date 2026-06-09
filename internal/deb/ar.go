package deb

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// arReader is a minimal parser for the BSD ar archive format used by .deb files.
type arReader struct {
	r   io.Reader
	buf [60]byte
	pos int64
}

func newArReader(r io.Reader) *arReader {
	return &arReader{r: r}
}

type arEntry struct {
	r    io.Reader
	size int64
}

func (e *arEntry) Read(p []byte) (int, error) { return e.r.Read(p) }

// Close drains any unread bytes so the parent arReader stays aligned for the next entry.
func (e *arEntry) Close() error {
	_, err := io.Copy(io.Discard, e.r)
	return err
}

// Next advances to the next entry in the archive.
// Returns the filename, a ReadCloser for the entry content, and any error.
// Returns io.EOF when the archive is exhausted.
func (a *arReader) Next() (string, io.ReadCloser, error) {
	if a.pos == 0 {
		// Read the global header.
		sig := make([]byte, 8)
		if _, err := io.ReadFull(a.r, sig); err != nil {
			return "", nil, err
		}
		if string(sig) != "!<arch>\n" {
			return "", nil, fmt.Errorf("not an ar archive")
		}
		a.pos = 8
	}

	// Each entry header is 60 bytes.
	hdr := a.buf[:]
	n, err := io.ReadFull(a.r, hdr)
	if n == 0 && err == io.EOF {
		return "", nil, io.EOF
	}
	if err != nil {
		return "", nil, err
	}
	a.pos += 60

	name := strings.TrimRight(string(hdr[0:16]), " /")
	sizeStr := strings.TrimSpace(string(hdr[48:58]))
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("ar header size: %w", err)
	}

	// Verify the magic bytes at bytes 58-59.
	magic := binary.BigEndian.Uint16(hdr[58:60])
	if magic != 0x600A {
		return "", nil, fmt.Errorf("ar header magic mismatch: %04x", magic)
	}

	lr := &io.LimitedReader{R: a.r, N: size}
	a.pos += size

	// Entries are 2-byte aligned; skip padding byte if needed.
	entry := &arEntry{
		r:    newPaddedReader(lr, size),
		size: size,
	}
	return name, entry, nil
}

// paddedReader wraps a LimitedReader and discards a padding byte when the
// underlying size is odd (ar archives pad entries to 2-byte boundaries).
type paddedReader struct {
	lr    *io.LimitedReader
	size  int64
	doneN int64
}

func newPaddedReader(lr *io.LimitedReader, size int64) io.Reader {
	return &paddedReader{lr: lr, size: size}
}

func (p *paddedReader) Read(buf []byte) (int, error) {
	n, err := p.lr.Read(buf)
	p.doneN += int64(n)
	if err == io.EOF && p.size%2 != 0 {
		// Discard the padding byte.
		pad := make([]byte, 1)
		p.lr.R.Read(pad) //nolint:errcheck
	}
	return n, err
}
