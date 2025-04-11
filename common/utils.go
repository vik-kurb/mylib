package common

import (
	"database/sql"
	"log"
	"time"
)

func ToNullTime(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(DateFormat, s)
	if err != nil {
		log.Print("Error while parsing date:", err)
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
