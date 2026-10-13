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
	PrevHash  string    `json:"prev_hash"`
	Hash      string    `json:"hash"`
}
