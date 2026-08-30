// Package postgres contains typed PostgreSQL repositories for Voidcut.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNilPool = errors.New("postgres repository requires a non-nil pool")

// ActivityRepository persists activity event envelopes.
type ActivityRepository struct {
	pool *pgxpool.Pool
}

// NewActivityRepository creates an activity repository backed by pool.
func NewActivityRepository(pool *pgxpool.Pool) (*ActivityRepository, error) {
	if pool == nil {
		return nil, errNilPool
	}
	return &ActivityRepository{pool: pool}, nil
}

// Insert stores one event. event_id is the primary key, so duplicate event
// delivery is rejected by PostgreSQL instead of creating a second record.
func (repository *ActivityRepository) Insert(ctx context.Context, event schema.EventEnvelope) error {
	if event.EventID == "" || event.TraceID == "" || event.UserID == "" || event.SessionID == "" || event.DeviceID == "" || event.EventType == "" || event.Version == "" || event.Timestamp.IsZero() {
		return errors.New("activity event is missing a required field")
	}
	if !json.Valid(event.Payload) {
		return errors.New("activity event payload is not valid JSON")
	}

	const query = `
		INSERT INTO activity_events (
			event_id, trace_id, user_id, session_id, device_id,
			event_type, version, event_timestamp, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := repository.pool.Exec(ctx, query,
		event.EventID,
		event.TraceID,
		event.UserID,
		event.SessionID,
		event.DeviceID,
		event.EventType,
		event.Version,
		event.Timestamp,
		event.Payload,
	)
	return err
}

// ListBySession returns events in timestamp order for one user's session.
func (repository *ActivityRepository) ListBySession(ctx context.Context, userID, sessionID string, from, to time.Time) ([]schema.EventEnvelope, error) {
	if userID == "" || sessionID == "" {
		return nil, errors.New("user ID and session ID are required")
	}

	const query = `
		SELECT event_id, trace_id, user_id, session_id, device_id,
			event_type, version, event_timestamp, payload
		FROM activity_events
		WHERE user_id = $1
		  AND session_id = $2
		  AND event_timestamp >= $3
		  AND event_timestamp < $4
		ORDER BY event_timestamp ASC, event_id ASC`

	rows, err := repository.pool.Query(ctx, query, userID, sessionID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []schema.EventEnvelope
	for rows.Next() {
		var event schema.EventEnvelope
		var payload []byte
		if err := rows.Scan(
			&event.EventID,
			&event.TraceID,
			&event.UserID,
			&event.SessionID,
			&event.DeviceID,
			&event.EventType,
			&event.Version,
			&event.Timestamp,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan activity event: %w", err)
		}
		event.Payload = json.RawMessage(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
