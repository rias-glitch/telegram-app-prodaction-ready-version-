package domain

import "time"

type Transaction struct {
	ID        int64                  `db:"id" json:"id"`
	UserID    int64                  `db:"user_id" json:"user_id"`
	Type      string                 `db:"type" json:"type"`
	Amount    int64                  `db:"amount" json:"amount"`
	Meta      map[string]interface{} `db:"meta" json:"meta,omitempty"`
	CreatedAt time.Time              `db:"created_at" json:"created_at"`
}
