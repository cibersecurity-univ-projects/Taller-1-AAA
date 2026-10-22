package models

import "time"

// AuditEntry es un registro de la bitácora de auditoría. Cada entrada se
// encadena con la anterior mediante PrevHash y se sella con Hash (HMAC).
type AuditEntry struct {
	Seq       int       `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	ActorID   int       `json:"actor_id"`
	ActorRole string    `json:"actor_role"`
	ClientIP  string    `json:"client_ip"`
	Method    string    `json:"method"`
	Operation string    `json:"operation"`
	Resource  string    `json:"resource"`
	Status    int       `json:"status"`
	// Email es el correo con que se intentó autenticar o registrar quien no
	// pudo ser identificado. Va vacío en el resto de operaciones.
	Email    string `json:"email,omitempty"`
	PrevHash string `json:"prev_hash"`
	Hash     string `json:"hash"`
}
