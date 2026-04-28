package io

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SQLModule provides SQL database functions for go-json programs.
type SQLModule struct {
	security *SecurityConfig
	config   map[string]any

	// Hosted mode: connection provided by host via WithSQLConnection.
	hostedDB *sql.DB

	mu sync.Mutex
	tx *sql.Tx
	sp int // savepoint counter for nested transactions
}

// NewSQLModule creates a new SQL I/O module in standalone mode.
func NewSQLModule(security *SecurityConfig) *SQLModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return &SQLModule{security: security}
}

// NewSQLModuleHosted creates a new SQL I/O module in hosted mode with a pre-configured connection.
func NewSQLModuleHosted(security *SecurityConfig, db *sql.DB) *SQLModule {
	m := NewSQLModule(security)
	m.hostedDB = db
	return m
}

func (m *SQLModule) Name() string { return "sql" }

func (m *SQLModule) SetConfig(cfg map[string]any) { m.config = cfg }

func (m *SQLModule) Functions() map[string]any {
	return map[string]any{
		"query":    m.sqlQuery,
		"execute":  m.sqlExecute,
		"begin":    m.sqlBegin,
		"commit":   m.sqlCommit,
		"rollback": m.sqlRollback,
	}
}

func (m *SQLModule) sqlQuery(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("sql.query: query is required")
	}

	query, args, dsn, err := m.parseQueryParams(params)
	if err != nil {
		return nil, err
	}

	db, err := m.getDB(dsn)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(m.security.SQL.MaxQueryTime) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var rows *sql.Rows
	m.mu.Lock()
	if m.tx != nil {
		rows, err = m.tx.QueryContext(ctx, query, args...)
	} else {
		rows, err = db.QueryContext(ctx, query, args...)
	}
	m.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}

	var result []any
	maxRows := m.security.SQL.MaxRows
	if maxRows <= 0 {
		maxRows = 10000
	}
	rowCount := 0

	for rows.Next() {
		if rowCount >= maxRows {
			break
		}

		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("sql.query: %s", err.Error())
		}

		row := make(map[string]any)
		for i, col := range columns {
			row[col] = convertSQLValue(values[i])
		}
		result = append(result, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.query: %s", err.Error())
	}

	if result == nil {
		result = []any{}
	}

	colsAny := make([]any, len(columns))
	for i, c := range columns {
		colsAny[i] = c
	}

	return map[string]any{
		"rows":    result,
		"columns": colsAny,
		"count":   rowCount,
	}, nil
}

func (m *SQLModule) sqlExecute(params ...any) (any, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("sql.execute: query is required")
	}

	query, args, dsn, err := m.parseQueryParams(params)
	if err != nil {
		return nil, err
	}

	db, err := m.getDB(dsn)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(m.security.SQL.MaxQueryTime) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var res sql.Result
	m.mu.Lock()
	if m.tx != nil {
		res, err = m.tx.ExecContext(ctx, query, args...)
	} else {
		res, err = db.ExecContext(ctx, query, args...)
	}
	m.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("sql.execute: %s", err.Error())
	}

	rowsAffected, _ := res.RowsAffected()
	lastInsertID, _ := res.LastInsertId()

	return map[string]any{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	}, nil
}

func (m *SQLModule) sqlBegin(params ...any) (any, error) {
	db, err := m.getDB("")
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx != nil {
		// Nested transaction — use savepoint.
		m.sp++
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("SAVEPOINT %s", spName))
		if err != nil {
			return nil, fmt.Errorf("sql.begin: %s", err.Error())
		}
		return nil, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("sql.begin: %s", err.Error())
	}
	m.tx = tx
	m.sp = 0
	return nil, nil
}

func (m *SQLModule) sqlCommit(params ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx == nil {
		return nil, fmt.Errorf("sql.commit: no active transaction")
	}

	if m.sp > 0 {
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", spName))
		m.sp--
		if err != nil {
			return nil, fmt.Errorf("sql.commit: %s", err.Error())
		}
		return nil, nil
	}

	err := m.tx.Commit()
	m.tx = nil
	m.sp = 0
	if err != nil {
		return nil, fmt.Errorf("sql.commit: %s", err.Error())
	}
	return nil, nil
}

func (m *SQLModule) sqlRollback(params ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tx == nil {
		return nil, fmt.Errorf("sql.rollback: no active transaction")
	}

	if m.sp > 0 {
		spName := fmt.Sprintf("sp_%d", m.sp)
		_, err := m.tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", spName))
		m.sp--
		if err != nil {
			return nil, fmt.Errorf("sql.rollback: %s", err.Error())
		}
		return nil, nil
	}

	err := m.tx.Rollback()
	m.tx = nil
	m.sp = 0
	if err != nil {
		return nil, fmt.Errorf("sql.rollback: %s", err.Error())
	}
	return nil, nil
}

func (m *SQLModule) parseQueryParams(params []any) (string, []any, string, error) {
	query, ok := params[0].(string)
	if !ok {
		return "", nil, "", fmt.Errorf("sql: query must be a string")
	}

	var args []any
	var dsn string

	for i := 1; i < len(params); i++ {
		switch v := params[i].(type) {
		case []any:
			args = v
		case map[string]any:
			if d, ok := v["dsn"].(string); ok {
				dsn = d
			}
			if a, ok := v["args"].([]any); ok {
				args = a
			}
		case string:
			dsn = v
		}
	}

	return query, args, dsn, nil
}

func (m *SQLModule) getDB(dsn string) (*sql.DB, error) {
	if m.hostedDB != nil {
		return m.hostedDB, nil
	}

	if dsn == "" {
		return nil, fmt.Errorf("sql: DSN is required in standalone mode")
	}

	driver := detectDriverFromDSN(dsn)
	if err := m.security.ValidateSQLDriver(driver); err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql: cannot open database: %s", err.Error())
	}

	return db, nil
}

func detectDriverFromDSN(dsn string) string {
	if len(dsn) > 11 && dsn[:11] == "postgres://" {
		return "postgres"
	}
	if len(dsn) > 8 && dsn[:8] == "mysql://" {
		return "mysql"
	}
	if len(dsn) > 10 && dsn[:10] == "sqlite3://" {
		return "sqlite3"
	}
	if len(dsn) > 5 && dsn[:5] == "file:" {
		return "sqlite3"
	}
	return "sqlite3"
}

func convertSQLValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return val
	case float64:
		return val
	case bool:
		return val
	case string:
		return val
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", val)
	}
}
