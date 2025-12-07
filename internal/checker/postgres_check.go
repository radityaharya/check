package checker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"gocheck/internal/models"
)

func (e *Engine) performPostgresCheck(check *models.Check, history *models.CheckHistory, start time.Time) {
	if check.PostgresConnString == "" {
		history.Success = false
		history.ErrorMessage = "no connection string specified"
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(check.TimeoutSeconds)*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", check.PostgresConnString)
	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("connection error: %v", err)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Duration(check.TimeoutSeconds) * time.Second)
	db.SetMaxOpenConns(1)

	if check.PostgresQuery == "" {
		err = db.PingContext(ctx)
		history.ResponseTimeMs = int(time.Since(start).Milliseconds())
		if err != nil {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("ping failed: %v", err)
			return
		}
		history.Success = true
		return
	}

	var result string
	err = db.QueryRowContext(ctx, check.PostgresQuery).Scan(&result)
	history.ResponseTimeMs = int(time.Since(start).Milliseconds())

	if err != nil {
		history.Success = false
		history.ErrorMessage = fmt.Sprintf("query failed: %v", err)
		return
	}

	history.ResponseBody = result

	if check.ExpectedQueryValue != "" {
		if result == check.ExpectedQueryValue {
			history.Success = true
		} else {
			history.Success = false
			history.ErrorMessage = fmt.Sprintf("expected '%s', got '%s'", check.ExpectedQueryValue, result)
		}
	} else {
		history.Success = true
	}
}
