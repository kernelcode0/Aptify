package deb

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"crypto/md5"  // #nosec G501
	"crypto/sha1" // #nosec G505
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// Info holds the parsed metadata from a .deb control file.
type Info struct {
	Package      string
	Version      string
	Architecture string
	Maintainer   string
	Description  string
	Depends      string
	Section      string
	Priority     string
	Homepage     string
	// Extra holds any additional control fields not captured above.
	Extra map[string]string

	// File properties computed from the raw bytes.
	Size   int64
	SHA256 string
	SHA1   string
	MD5    string

	// Raw control paragraph (used verbatim in Packages index).
	ControlBlock string
}

// Parse reads a .deb file from r and returns its metadata.
// It also computes checksums and size over the raw bytes.
func Parse(r io.Reader) (*Info, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read deb: %w", err)
	}

	info := &Info{
		Extra: make(map[string]string),
		Size:  int64(len(raw)),
	}

	// Compute checksums.
	s256 := sha256.Sum256(raw)
	s1 := sha1.Sum(raw) // #nosec G401 -- Required by APT format
	m5 := md5.Sum(raw)  // #nosec G401 -- Required by APT format
	info.SHA256 = hex.EncodeToString(s256[:])
	info.SHA1 = hex.EncodeToString(s1[:])
	info.MD5 = hex.EncodeToString(m5[:])

	control, err := extractControl(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("extract control: %w", err)
	}
	info.ControlBlock = control
	if err := parseControl(info, control); err != nil {
		return nil, err
	}
	return info, nil
}

// extractControl reads the ar archive and returns the raw content of the
// control file found inside control.tar.{gz,bz2,xz}.
func extractControl(r io.ReaderAt) (string, error) {
	size, err := r.(interface{ Seek(int64, int) (int64, error) }).Seek(0, io.SeekEnd)
	if err != nil {
		// Fall back: read all into buffer to determine size.
		return "", fmt.Errorf("seek: %w", err)
	}

	ar := newArReader(io.NewSectionReader(r, 0, size))
	for {
		name, rc, err := ar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("ar: %w", err)
		}
		if strings.HasPrefix(name, "control.tar") {
			ctrl, err := extractControlFromTar(name, rc)
			_ = rc.Close()
			return ctrl, err
		}
		_ = rc.Close()
	}
	return "", fmt.Errorf("control.tar not found in .deb archive")
}

func extractControlFromTar(name string, r io.Reader) (string, error) {
	var tr *tar.Reader
	switch {
	case strings.HasSuffix(name, ".gz"):
		gz, err := gzip.NewReader(r)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		tr = tar.NewReader(gz)
	case strings.HasSuffix(name, ".bz2"):
		tr = tar.NewReader(bzip2.NewReader(r))
	case strings.HasSuffix(name, ".xz"):
		xr, err := xz.NewReader(r)
		if err != nil {
			return "", err
		}
		tr = tar.NewReader(xr)
	case strings.HasSuffix(name, ".zst"):
		zr, err := zstd.NewReader(r)
		if err != nil {
			return "", err
		}
		defer zr.Close()
		tr = tar.NewReader(zr)
	default:
		// uncompressed
		tr = tar.NewReader(r)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		base := strings.TrimPrefix(hdr.Name, "./")
		if base == "control" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("control file not found in control.tar")
}

func parseControl(info *Info, block string) error {
	scanner := bufio.NewScanner(strings.NewReader(block))
	var currentKey string
	var currentVal strings.Builder

	flush := func() {
		if currentKey == "" {
			return
		}
		val := strings.TrimSpace(currentVal.String())
		switch strings.ToLower(currentKey) {
		case "package":
			info.Package = val
		case "version":
			info.Version = val
		case "architecture":
			info.Architecture = val
		case "maintainer":
			info.Maintainer = val
		case "description":
			info.Description = val
		case "depends":
			info.Depends = val
		case "section":
			info.Section = val
		case "priority":
			info.Priority = val
		case "homepage":
			info.Homepage = val
		default:
			info.Extra[currentKey] = val
		}
		currentKey = ""
		currentVal.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Continuation line.
			currentVal.WriteByte('\n')
			currentVal.WriteString(line)
			continue
		}
		flush()
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		currentKey = strings.TrimSpace(line[:idx])
		currentVal.WriteString(strings.TrimSpace(line[idx+1:]))
	}
	flush()

	if info.Package == "" {
		return fmt.Errorf("control file missing Package field")
	}
	return nil
}
