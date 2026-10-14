// Package storage persiste colecciones en archivos JSON usando solo la
// librería estándar. Cada Table es segura para uso concurrente.
package storage

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

var (
	ErrNotFound = errors.New("registro no encontrado")
	ErrConflict = errors.New("registro en conflicto con uno existente")
)

type tableData[T any] struct {
	NextID int       `json:"next_id"`
	Rows   map[int]T `json:"rows"`
}

type Table[T any] struct {
	mu   sync.RWMutex
	path string
	data tableData[T]
}

// OpenTable carga la tabla desde path o la crea vacía si el archivo no existe.
func OpenTable[T any](path string) (*Table[T], error) {
	t := &Table[T]{path: path, data: tableData[T]{NextID: 1, Rows: map[int]T{}}}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &t.data); err != nil {
		return nil, err
	}
	return t, nil
}

// Insert construye un registro con el siguiente ID y lo guarda. Si conflicts
// no es nil y coincide con algún registro existente, devuelve ErrConflict.
func (t *Table[T]) Insert(build func(id int) T, conflicts func(T) bool) (T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var zero T
	if conflicts != nil {
		for _, row := range t.data.Rows {
			if conflicts(row) {
				return zero, ErrConflict
			}
		}
	}

	row := build(t.data.NextID)
	t.data.Rows[t.data.NextID] = row
	t.data.NextID++
	if err := t.persist(); err != nil {
		return zero, err
	}
	return row, nil
}

func (t *Table[T]) Get(id int) (T, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	row, ok := t.data.Rows[id]
	if !ok {
		return row, ErrNotFound
	}
	return row, nil
}

// Find devuelve el primer registro (en orden de ID) que cumple match.
func (t *Table[T]) Find(match func(T) bool) (T, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, id := range t.sortedIDs() {
		if row := t.data.Rows[id]; match(row) {
			return row, nil
		}
	}
	var zero T
	return zero, ErrNotFound
}

// Filter devuelve, en orden de ID, los registros que cumplen match.
func (t *Table[T]) Filter(match func(T) bool) []T {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rows := []T{}
	for _, id := range t.sortedIDs() {
		if row := t.data.Rows[id]; match(row) {
			rows = append(rows, row)
		}
	}
	return rows
}

// Update aplica change sobre una copia del registro y la guarda solo si
// change no devuelve error.
func (t *Table[T]) Update(id int, change func(*T) error) (T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	row, ok := t.data.Rows[id]
	if !ok {
		return row, ErrNotFound
	}
	if err := change(&row); err != nil {
		return row, err
	}
	t.data.Rows[id] = row
	return row, t.persist()
}

func (t *Table[T]) Delete(id int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.data.Rows[id]; !ok {
		return ErrNotFound
	}
	delete(t.data.Rows, id)
	return t.persist()
}

// DeleteWhere elimina todos los registros que cumplen match.
func (t *Table[T]) DeleteWhere(match func(T) bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	removed := false
	for id, row := range t.data.Rows {
		if match(row) {
			delete(t.data.Rows, id)
			removed = true
		}
	}
	if !removed {
		return nil
	}
	return t.persist()
}

func (t *Table[T]) sortedIDs() []int {
	return slices.Sorted(maps.Keys(t.data.Rows))
}

// persist escribe en un archivo temporal y lo renombra para no dejar el
// archivo a medias si el proceso se interrumpe.
func (t *Table[T]) persist() error {
	raw, err := json.MarshalIndent(t.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o700); err != nil {
		return err
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, t.path)
}
