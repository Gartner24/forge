package secrets

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Store is an encrypted key-value store backed by a single age-encrypted file.
type Store struct {
	path      string
	identity  *age.X25519Identity
	recipient *age.X25519Recipient
}

// New opens or creates an encrypted secrets store at the given path.
// If the store does not exist, it creates a new one with a fresh key.
func New(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating secrets directory: %w", err)
	}

	keyPath := path + ".key"

	var identity *age.X25519Identity

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// No key yet — generate a new one and save it.
		identity, err = age.GenerateX25519Identity()
		if err != nil {
			return nil, fmt.Errorf("generating key: %w", err)
		}
		if err := os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0600); err != nil {
			return nil, fmt.Errorf("saving key: %w", err)
		}
	} else {
		// Load the existing key.
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("reading key: %w", err)
		}
		identity, err = age.ParseX25519Identity(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, fmt.Errorf("parsing key: %w", err)
		}
	}

	return &Store{
		path:      path,
		identity:  identity,
		recipient: identity.Recipient(),
	}, nil
}

// Set stores a key-value pair in the secrets store.
// Keys are namespaced with dots, e.g. "smeltforge.myapp.DATABASE_URL".
func (s *Store) Set(key, value string) error {
	data, err := s.readAll()
	if err != nil {
		return err
	}

	data[key] = value

	return s.writeAll(data)
}

// Get retrieves a value by key. Returns an error if the key does not exist.
func (s *Store) Get(key string) (string, error) {
	data, err := s.readAll()
	if err != nil {
		return "", err
	}

	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}

	return val, nil
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) error {
	data, err := s.readAll()
	if err != nil {
		return err
	}

	delete(data, key)

	return s.writeAll(data)
}

// List returns all keys in the store.
func (s *Store) List() ([]string, error) {
	data, err := s.readAll()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	return keys, nil
}

// readAll decrypts and parses the secrets file into a map.
// Returns an empty map if the file does not exist yet.
func (s *Store) readAll() (map[string]string, error) {
	data := make(map[string]string)

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return data, nil
	}

	ciphertext, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("reading secrets file: %w", err)
	}

	armorReader := armor.NewReader(bytes.NewReader(ciphertext))
	r, err := age.Decrypt(armorReader, s.identity)
	if err != nil {
		return nil, fmt.Errorf("decrypting secrets: %w", err)
	}

	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading decrypted secrets: %w", err)
	}

	// Format is one "key=value" per line.
	for _, line := range strings.Split(string(plain), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		k, v := line[:idx], line[idx+1:]
		data[k] = v
	}

	return data, nil
}

// writeAll encrypts and writes the full secrets map to disk.
func (s *Store) writeAll(data map[string]string) error {
	var plain strings.Builder
	for k, v := range data {
		plain.WriteString(k + "=" + v + "\n")
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)

	w, err := age.Encrypt(armorWriter, s.recipient)
	if err != nil {
		return fmt.Errorf("starting encryption: %w", err)
	}

	if _, err := io.WriteString(w, plain.String()); err != nil {
		return fmt.Errorf("writing secrets: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("finalizing encryption: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return fmt.Errorf("finalizing armor: %w", err)
	}

	return os.WriteFile(s.path, buf.Bytes(), 0600)
}
