package user

import (
	"database/sql"
	"time"

	"github.com/fujidaiti/paperdoll/server/infra"
)

type Service struct {
	DB        *sql.DB
	Now       func() time.Time
	SendEmail infra.EmailSender
}

func NewService(db *sql.DB, es infra.EmailSender) *Service {
	return &Service{
		DB:        db,
		Now:       func() time.Time { return time.Now() },
		SendEmail: es,
	}
}
