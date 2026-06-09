package signing

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	pgp "github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// Signer holds a loaded private key and can sign Release files.
type Signer struct {
	entity *pgp.Entity
}

// LoadOrGenerate loads the private key from keyPath, or generates a new one
// if the file does not exist yet. The public key is written to pubPath.
func LoadOrGenerate(keyPath, pubPath, name, email string) (*Signer, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return generate(keyPath, pubPath, name, email)
	}
	return load(keyPath)
}

func generate(keyPath, pubPath, name, email string) (*Signer, error) {
	cfg := &packet.Config{
		RSABits: 4096,
		Time:    func() time.Time { return time.Now() },
	}
	entity, err := pgp.NewEntity(name, "APT Repository Signing Key", email, cfg)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	// Write private key.
	var privBuf bytes.Buffer
	w, err := armor.Encode(&privBuf, "PGP PRIVATE KEY BLOCK", nil)
	if err != nil {
		return nil, err
	}
	if err := entity.SerializePrivate(w, nil); err != nil {
		return nil, err
	}
	_ = w.Close()
	if err := os.WriteFile(keyPath, privBuf.Bytes(), 0600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	// Write public key.
	if err := writePublicKey(pubPath, entity); err != nil {
		return nil, err
	}

	return &Signer{entity: entity}, nil
}

func load(keyPath string) (*Signer, error) {
	// #nosec G304 -- keyPath is generated internally by the application
	f, err := os.Open(keyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	block, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("armor decode: %w", err)
	}
	entities, err := pgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("read key ring: %w", err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("no keys found in %s", keyPath)
	}
	return &Signer{entity: entities[0]}, nil
}

// ClearSign produces an OpenPGP clear-signed version of data (used for InRelease).
func (s *Signer) ClearSign(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := clearsign.Encode(&buf, s.entity.PrivateKey, nil)
	if err != nil {
		return nil, fmt.Errorf("clearsign: %w", err)
	}
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DetachSign produces a detached ASCII-armored signature (used for Release.gpg).
func (s *Signer) DetachSign(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := pgp.ArmoredDetachSign(&buf, s.entity, bytes.NewReader(data), nil); err != nil {
		return nil, fmt.Errorf("detach sign: %w", err)
	}
	return buf.Bytes(), nil
}

// PublicKeyArmored returns the public key in ASCII-armored format.
func (s *Signer) PublicKeyArmored() ([]byte, error) {
	var buf bytes.Buffer
	pw, err := armor.Encode(&buf, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return nil, err
	}
	if err := s.entity.Serialize(pw); err != nil {
		return nil, err
	}
	_ = pw.Close()
	return buf.Bytes(), nil
}

// PrivateKeyArmored returns the private key in ASCII-armored format.
func (s *Signer) PrivateKeyArmored() ([]byte, error) {
	var buf bytes.Buffer
	pw, err := armor.Encode(&buf, "PGP PRIVATE KEY BLOCK", nil)
	if err != nil {
		return nil, err
	}
	if err := s.entity.SerializePrivate(pw, nil); err != nil {
		return nil, err
	}
	_ = pw.Close()
	return buf.Bytes(), nil
}

func writePublicKey(pubPath string, entity *pgp.Entity) error {
	var pubBuf bytes.Buffer
	pw, err := armor.Encode(&pubBuf, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return err
	}
	if err := entity.Serialize(pw); err != nil {
		return err
	}
	_ = pw.Close()
	if pubPath != "" {
		// #nosec G306 -- Public keys are meant to be world-readable
		return os.WriteFile(pubPath, pubBuf.Bytes(), 0644)
	}
	return nil
}
