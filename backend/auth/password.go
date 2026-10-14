// Package auth reúne los mecanismos de autenticación: hash de contraseñas,
// sesiones por token y la identidad del solicitante dentro del request.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	hashScheme       = "pbkdf2-sha256"
	hashIterations   = 600_000 // recomendación OWASP para PBKDF2-HMAC-SHA256
	hashSaltBytes    = 16
	hashKeyBytes     = 32
	hashFieldsLength = 4
)

var b64 = base64.RawStdEncoding

// HashPassword deriva la contraseña con PBKDF2 y una sal aleatoria. El
// resultado se guarda como "esquema$iteraciones$sal$clave".
func HashPassword(password string) (string, error) {
	salt := make([]byte, hashSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, hashIterations, hashKeyBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%d$%s$%s", hashScheme, hashIterations, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword recalcula la derivación con la sal almacenada y compara en
// tiempo constante.
func VerifyPassword(password, encoded string) bool {
	fields := strings.Split(encoded, "$")
	if len(fields) != hashFieldsLength || fields[0] != hashScheme {
		return false
	}
	iterations, err := strconv.Atoi(fields[1])
	if err != nil {
		return false
	}
	salt, err := b64.DecodeString(fields[2])
	if err != nil {
		return false
	}
	expected, err := b64.DecodeString(fields[3])
	if err != nil {
		return false
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(expected))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(key, expected) == 1
}
