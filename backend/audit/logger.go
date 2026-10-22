// Package audit implementa la bitácora de auditoría (Accounting). Las
// entradas se agregan a un archivo JSON Lines y forman una cadena: cada una
// incluye el hash de la anterior y se firma con HMAC-SHA256. Modificar,
// borrar o insertar una entrada rompe la cadena y se detecta al verificarla.
package audit

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"automarket/models"
)

var ErrTampered = errors.New("la bitácora de auditoría fue alterada")

var genesisHash = strings.Repeat("0", sha256.Size*2)

type Logger struct {
	mu       sync.Mutex
	file     *os.File
	key      []byte
	seq      int
	lastHash string
}

// Open verifica la integridad de la bitácora existente y la deja lista para
// agregar nuevas entradas a continuación de la última.
func Open(path string, key []byte) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}

	l := &Logger{file: file, key: key, lastHash: genesisHash}
	if err := l.verify(); err != nil {
		file.Close()
		return nil, err
	}
	return l, nil
}

// Record completa la entrada con su posición en la cadena, la firma y la
// escribe de forma durable.
func (l *Logger) Record(entry models.AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.Seq = l.seq + 1
	entry.Timestamp = time.Now().UTC()
	entry.PrevHash = l.lastHash
	hash, err := l.sign(entry)
	if err != nil {
		return err
	}
	entry.Hash = hash

	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := l.file.Write(append(line, '\n')); err != nil {
		return err
	}
	if err := l.file.Sync(); err != nil {
		return err
	}
	l.seq, l.lastHash = entry.Seq, entry.Hash
	return nil
}

func (l *Logger) Close() error {
	return l.file.Close()
}

// verify recorre la bitácora desde el inicio comprobando secuencia, enlace
// con la entrada anterior y firma de cada entrada.
func (l *Logger) verify() error {
	scanner := bufio.NewScanner(l.file)
	for scanner.Scan() {
		var entry models.AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return fmt.Errorf("%w: línea %d ilegible", ErrTampered, l.seq+1)
		}
		expected, err := l.sign(entry)
		if err != nil {
			return err
		}
		if entry.Seq != l.seq+1 || entry.PrevHash != l.lastHash || !hmac.Equal([]byte(entry.Hash), []byte(expected)) {
			return fmt.Errorf("%w: entrada %d", ErrTampered, l.seq+1)
		}
		l.seq, l.lastHash = entry.Seq, entry.Hash
	}
	return scanner.Err()
}

// sign calcula el HMAC de la entrada sin su propio campo Hash.
func (l *Logger) sign(entry models.AuditEntry) (string, error) {
	entry.Hash = ""
	payload, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, l.key)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
