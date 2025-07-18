package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourname/sleeptracker/internal"
)

type PostgresStorage struct {
	pool   *pgxpool.Pool
	logger internal.Logger
}

func NewPostgresStorage(dsn string, logger internal.Logger) (*PostgresStorage, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Errorf("failed to connect to postgres: %v", err)
		return nil, err
	}
	return &PostgresStorage{pool: pool, logger: logger}, nil
}

// --- SleepLogRepository ---
func (p *PostgresStorage) SaveSleepLog(ctx context.Context, log *internal.SleepLog) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO sleep_logs (id, user_id, start_time, end_time, quality, reason, interruptions, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		log.ID, log.UserID, log.StartTime, log.EndTime, log.Quality, log.Reason, log.Interruptions, log.CreatedAt)
	if err != nil {
		p.logger.Errorf("failed to insert sleep log: %v", err)
		return err
	}
	return nil
}

func (p *PostgresStorage) ListSleepLogs(ctx context.Context, userID string) ([]internal.SleepLog, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, user_id, start_time, end_time, quality, reason, interruptions, created_at FROM sleep_logs WHERE user_id = $1 ORDER BY start_time DESC`, userID)
	if err != nil {
		p.logger.Errorf("failed to query sleep logs: %v", err)
		return nil, err
	}
	defer rows.Close()

	var logs []internal.SleepLog
	for rows.Next() {
		var l internal.SleepLog
		err := rows.Scan(&l.ID, &l.UserID, &l.StartTime, &l.EndTime, &l.Quality, &l.Reason, &l.Interruptions, &l.CreatedAt)
		if err != nil {
			p.logger.Errorf("failed to scan sleep log: %v", err)
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// --- GoalRepository ---
func (p *PostgresStorage) SetGoal(ctx context.Context, goal *internal.Goal) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO goals (id, user_id, type, value, created_at) VALUES ($1, $2, $3, $4, $5)`,
		goal.ID, goal.UserID, goal.Type, goal.Value, goal.CreatedAt)
	if err != nil {
		p.logger.Errorf("failed to insert goal: %v", err)
		return err
	}
	return nil
}

func (p *PostgresStorage) GetGoal(ctx context.Context, userID string) (*internal.Goal, error) {
	row := p.pool.QueryRow(ctx, `SELECT id, user_id, type, value, created_at FROM goals WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID)
	var g internal.Goal
	if err := row.Scan(&g.ID, &g.UserID, &g.Type, &g.Value, &g.CreatedAt); err != nil {
		p.logger.Errorf("goal not found: %v", err)
		return nil, err
	}
	return &g, nil
}

// --- UserRepository ---
func (p *PostgresStorage) GetUserByToken(ctx context.Context, token string) (*internal.User, error) {
	row := p.pool.QueryRow(ctx, `SELECT id, token, name FROM users WHERE token = $1`, token)
	var u internal.User
	if err := row.Scan(&u.ID, &u.Token, &u.Name); err != nil {
		p.logger.Errorf("user not found: %v", err)
		return nil, err
	}
	return &u, nil
}

// --- Compile-time assertions ---
var _ SleepLogRepository = (*PostgresStorage)(nil)
var _ GoalRepository = (*PostgresStorage)(nil)
