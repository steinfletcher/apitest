package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/steinfletcher/apitest"
)

// WrapWithRecorder wraps an existing driver with a Recorder
func WrapWithRecorder(driverName string, recorder *apitest.Recorder) driver.Driver {
	sqlDriver := sqlDriverNameToDriver(driverName)
	recordingDriver := &RecordingDriver{
		sourceName: driverName,
		Driver:     sqlDriver,
		recorder:   recorder,
	}

	if _, ok := sqlDriver.(driver.DriverContext); ok {
		return &RecordingDriverContext{recordingDriver}
	}

	return recordingDriver
}

type RecordingDriver struct {
	Driver     driver.Driver
	recorder   *apitest.Recorder
	sourceName string
}

func (d *RecordingDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	_, isConnQuery := conn.(driver.Queryer)
	_, isConnQueryCtx := conn.(driver.QueryerContext)
	_, isConnExec := conn.(driver.Execer)
	_, isConnExecCtx := conn.(driver.ExecerContext)
	_, isConnPrepareCtx := conn.(driver.ConnPrepareContext)
	recordingConn := &RecordingConn{Conn: conn, recorder: d.recorder, sourceName: d.sourceName}

	if isConnQueryCtx && isConnExecCtx && isConnPrepareCtx {
		return &RecordingConnWithExecQueryPrepareContext{
			recordingConn,
			&RecordingConnWithPrepareContext{recordingConn},
			&RecordingConnWithExecContext{recordingConn},
			&RecordingConnWithQueryContext{recordingConn},
			&RecordingConnWithBeginTx{recordingConn},
			&RecordingConnWithPing{recordingConn},
		}, nil
	}

	if isConnQuery && isConnExec {
		return &RecordingConnWithExecQuery{
			recordingConn,
			&RecordingConnWithExec{recordingConn},
			&RecordingConnWithQuery{recordingConn},
		}, nil
	}

	return recordingConn, nil
}

type RecordingDriverContext struct {
	*RecordingDriver
}

func (d *RecordingDriverContext) OpenConnector(name string) (driver.Connector, error) {
	if driverCtx, ok := d.Driver.(driver.DriverContext); ok {
		connector, err := driverCtx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &RecordingConnector{recorder: d.recorder, sourceName: d.sourceName, Connector: connector}, nil
	}

	return nil, errors.New("OpenConnector not implemented")
}

type RecordingConnector struct {
	Connector  driver.Connector
	recorder   *apitest.Recorder
	sourceName string
}

func (c *RecordingConnector) Connect(context context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(context)
	if err != nil {
		return nil, err
	}

	_, isConnQuery := conn.(driver.Queryer)
	_, isConnQueryCtx := conn.(driver.QueryerContext)
	_, isConnExec := conn.(driver.Execer)
	_, isConnExecCtx := conn.(driver.ExecerContext)
	_, isConnPrepareCtx := conn.(driver.ConnPrepareContext)
	recordingConn := &RecordingConn{Conn: conn, recorder: c.recorder, sourceName: c.sourceName}

	if isConnQueryCtx && isConnExecCtx && isConnPrepareCtx {
		return &RecordingConnWithExecQueryPrepareContext{
			recordingConn,
			&RecordingConnWithPrepareContext{recordingConn},
			&RecordingConnWithExecContext{recordingConn},
			&RecordingConnWithQueryContext{recordingConn},
			&RecordingConnWithBeginTx{recordingConn},
			&RecordingConnWithPing{recordingConn},
		}, nil
	}

	if isConnQuery && isConnExec {
		return &RecordingConnWithExecQuery{
			recordingConn,
			&RecordingConnWithExec{recordingConn},
			&RecordingConnWithQuery{recordingConn},
		}, nil
	}

	return recordingConn, nil
}

func (c *RecordingConnector) Driver() driver.Driver { return c.Connector.Driver() }

type RecordingConn struct {
	Conn       driver.Conn
	recorder   *apitest.Recorder
	sourceName string
}

func (conn *RecordingConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := conn.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}

	_, isStmtQueryContext := stmt.(driver.StmtQueryContext)
	_, isStmtExecContext := stmt.(driver.StmtExecContext)
	recordingStmt := &RecordingStmt{
		Stmt:       stmt,
		recorder:   conn.recorder,
		query:      query,
		sourceName: conn.sourceName,
	}

	if isStmtQueryContext && isStmtExecContext {
		return &RecordingStmtWithExecQueryContext{
			recordingStmt,
			&RecordingStmtWithExecContext{recordingStmt},
			&RecordingStmtWithQueryContext{recordingStmt},
		}, nil
	}

	return recordingStmt, nil
}

func (conn *RecordingConn) Close() error              { return conn.Conn.Close() }
func (conn *RecordingConn) Begin() (driver.Tx, error) { return conn.Conn.Begin() }

type RecordingConnWithQuery struct {
	*RecordingConn
}

func (conn *RecordingConnWithQuery) Query(query string, args []driver.Value) (driver.Rows, error) {
	if connQuery, ok := conn.Conn.(driver.Queryer); ok {
		rows, err := connQuery.Query(query, args)
		if err != nil {
			return nil, err
		}

		if conn.recorder != nil {
			recorderBody := query
			if len(args) > 0 {
				recorderBody = fmt.Sprintf("%s %+v", query, args)
			}
			conn.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    conn.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		return &RecordingRows{Rows: rows, recorder: conn.recorder, sourceName: conn.sourceName}, err
	}

	return nil, errors.New("Queryer not implemented")
}

type RecordingConnWithQueryContext struct {
	*RecordingConn
}

func (conn *RecordingConnWithQueryContext) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if connQueryCtx, ok := conn.Conn.(driver.QueryerContext); ok {
		rows, err := connQueryCtx.QueryContext(ctx, query, args)
		if err != nil {
			return nil, err
		}

		if conn.recorder != nil {
			recorderBody := query
			if len(args) > 0 {
				convertedArgs, convertErr := namedValueToValue(args)
				if convertErr != nil {
					return nil, convertErr
				}
				recorderBody = fmt.Sprintf("%s %+v", query, convertedArgs)
			}
			conn.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    conn.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		return &RecordingRows{Rows: rows, recorder: conn.recorder, sourceName: conn.sourceName}, err
	}

	return nil, errors.New("QueryerContext not implemented")
}

type RecordingConnWithExec struct {
	*RecordingConn
}

func (conn *RecordingConnWithExec) Exec(query string, args []driver.Value) (driver.Result, error) {
	if connExec, ok := conn.Conn.(driver.Execer); ok {
		result, err := connExec.Exec(query, args)
		if err != nil {
			return nil, err
		}

		if conn.recorder != nil {
			recorderBody := query
			if len(args) > 0 {
				recorderBody = fmt.Sprintf("%s %+v", query, args)
			}
			conn.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    conn.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		if result != nil && conn.recorder != nil {
			rowsAffected, _ := result.RowsAffected()
			conn.recorder.AddMessageResponse(apitest.MessageResponse{
				Source:    conn.sourceName,
				Target:    apitest.SystemUnderTestDefaultName,
				Header:    "SQL Result",
				Body:      fmt.Sprintf("Affected rows: %d", rowsAffected),
				Timestamp: time.Now().UTC(),
			})
		}

		return result, err
	}

	return nil, errors.New("Execer not implemented")
}

type RecordingConnWithExecContext struct {
	*RecordingConn
}

func (conn *RecordingConnWithExecContext) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if connExecCtx, ok := conn.Conn.(driver.ExecerContext); ok {
		result, err := connExecCtx.ExecContext(ctx, query, args)
		if err != nil {
			return nil, err
		}

		if conn.recorder != nil {
			recorderBody := query
			if len(args) > 0 {
				convertedArgs, convertErr := namedValueToValue(args)
				if convertErr != nil {
					return nil, convertErr
				}
				recorderBody = fmt.Sprintf("%s %+v", query, convertedArgs)
			}
			conn.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    conn.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		if result != nil && conn.recorder != nil {
			rowsAffected, _ := result.RowsAffected()
			conn.recorder.AddMessageResponse(apitest.MessageResponse{
				Source:    conn.sourceName,
				Target:    apitest.SystemUnderTestDefaultName,
				Header:    "SQL Result",
				Body:      fmt.Sprintf("Affected rows: %d", rowsAffected),
				Timestamp: time.Now().UTC(),
			})
		}

		return result, err
	}

	return nil, errors.New("ExecerContext not implemented")
}

type RecordingConnWithPrepareContext struct {
	*RecordingConn
}

func (conn *RecordingConnWithPrepareContext) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if connPrepareCtx, ok := conn.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := connPrepareCtx.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}

		_, isStmtQueryContext := stmt.(driver.StmtQueryContext)
		_, isStmtExecContext := stmt.(driver.StmtExecContext)
		recordingStmt := &RecordingStmt{Stmt: stmt, recorder: conn.recorder, query: query, sourceName: conn.sourceName}

		if isStmtQueryContext && isStmtExecContext {
			return &RecordingStmtWithExecQueryContext{
				recordingStmt,
				&RecordingStmtWithExecContext{recordingStmt},
				&RecordingStmtWithQueryContext{recordingStmt},
			}, nil
		}

		if isStmtQueryContext {
			return &RecordingStmtWithQueryContext{recordingStmt}, nil
		}

		if isStmtExecContext {
			return &RecordingStmtWithExecContext{recordingStmt}, nil
		}

		return recordingStmt, nil

	}

	return nil, errors.New("ConnPrepareContext not implemented")
}

type RecordingConnWithBeginTx struct {
	*RecordingConn
}

func (conn *RecordingConnWithBeginTx) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if connBeginTx, ok := conn.Conn.(driver.ConnBeginTx); ok {
		return connBeginTx.BeginTx(ctx, opts)
	}

	return nil, errors.New("ConnBeginTx not implemented")
}

type RecordingConnWithPing struct {
	*RecordingConn
}

func (conn *RecordingConnWithPing) Ping(ctx context.Context) error {
	if connPinger, ok := conn.Conn.(driver.Pinger); ok {
		return connPinger.Ping(ctx)
	}

	return errors.New("Pinger not implemented")
}

type RecordingConnWithExecQuery struct {
	*RecordingConn
	*RecordingConnWithExec
	*RecordingConnWithQuery
}

type RecordingConnWithExecQueryPrepareContext struct {
	*RecordingConn
	*RecordingConnWithPrepareContext
	*RecordingConnWithExecContext
	*RecordingConnWithQueryContext
	*RecordingConnWithBeginTx
	*RecordingConnWithPing
}

type RecordingStmt struct {
	Stmt       driver.Stmt
	recorder   *apitest.Recorder
	sourceName string
	query      string
}

func (stmt *RecordingStmt) Close() error {
	return stmt.Stmt.Close()
}

func (stmt *RecordingStmt) NumInput() int {
	return stmt.Stmt.NumInput()
}

func (stmt *RecordingStmt) Exec(args []driver.Value) (driver.Result, error) {
	result, err := stmt.Stmt.Exec(args)
	if stmt.recorder != nil {
		recorderBody := stmt.query
		if len(args) > 0 {
			recorderBody = fmt.Sprintf("%s %+v", stmt.query, args)
		}
		stmt.recorder.AddMessageRequest(apitest.MessageRequest{
			Source:    apitest.SystemUnderTestDefaultName,
			Target:    stmt.sourceName,
			Header:    "SQL Query",
			Body:      recorderBody,
			Timestamp: time.Now().UTC(),
		})
	}

	if result != nil && stmt.recorder != nil {
		rowsAffected, _ := result.RowsAffected()
		stmt.recorder.AddMessageResponse(apitest.MessageResponse{
			Source:    stmt.sourceName,
			Target:    apitest.SystemUnderTestDefaultName,
			Header:    "SQL Result",
			Body:      fmt.Sprintf("Affected rows: %d", rowsAffected),
			Timestamp: time.Now().UTC(),
		})
	}

	return result, err
}

func (stmt *RecordingStmt) Query(args []driver.Value) (driver.Rows, error) {
	rows, err := stmt.Stmt.Query(args)

	if stmt.recorder != nil {
		recorderBody := stmt.query
		if len(args) > 0 {
			recorderBody = fmt.Sprintf("%s %+v", stmt.query, args)
		}
		stmt.recorder.AddMessageRequest(apitest.MessageRequest{
			Source:    apitest.SystemUnderTestDefaultName,
			Target:    stmt.sourceName,
			Header:    "SQL Query",
			Body:      recorderBody,
			Timestamp: time.Now().UTC(),
		})
	}

	return &RecordingRows{Rows: rows, recorder: stmt.recorder, sourceName: stmt.sourceName}, err
}

type RecordingStmtWithExecContext struct {
	*RecordingStmt
}

func (stmt *RecordingStmtWithExecContext) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if stmtExecCtx, ok := stmt.Stmt.(driver.StmtExecContext); ok {
		result, err := stmtExecCtx.ExecContext(ctx, args)
		if err != nil {
			return nil, err
		}

		if stmt.recorder != nil {
			recorderBody := stmt.query
			if len(args) > 0 {
				convertedArgs, convertErr := namedValueToValue(args)
				if convertErr != nil {
					return nil, convertErr
				}
				recorderBody = fmt.Sprintf("%s %+v", stmt.query, convertedArgs)
			}

			stmt.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    stmt.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		if result != nil && stmt.recorder != nil {
			rowsAffected, _ := result.RowsAffected()
			stmt.recorder.AddMessageResponse(apitest.MessageResponse{
				Source:    stmt.sourceName,
				Target:    apitest.SystemUnderTestDefaultName,
				Header:    "SQL Result",
				Body:      fmt.Sprintf("Affected rows: %d", rowsAffected),
				Timestamp: time.Now().UTC(),
			})
		}

		return result, err
	}

	return nil, errors.New("StmtExecContext not implemented")
}

type RecordingStmtWithQueryContext struct {
	*RecordingStmt
}

func (stmt *RecordingStmtWithQueryContext) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if stmtQueryCtx, ok := stmt.Stmt.(driver.StmtQueryContext); ok {
		rows, err := stmtQueryCtx.QueryContext(ctx, args)
		if err != nil {
			return nil, err
		}

		if stmt.recorder != nil {
			recorderBody := stmt.query
			if len(args) > 0 {
				convertedArgs, convertErr := namedValueToValue(args)
				if convertErr != nil {
					return nil, convertErr
				}
				recorderBody = fmt.Sprintf("%s %+v", stmt.query, convertedArgs)
			}

			stmt.recorder.AddMessageRequest(apitest.MessageRequest{
				Source:    apitest.SystemUnderTestDefaultName,
				Target:    stmt.sourceName,
				Header:    "SQL Query",
				Body:      recorderBody,
				Timestamp: time.Now().UTC(),
			})
		}

		return &RecordingRows{Rows: rows, recorder: stmt.recorder, sourceName: stmt.sourceName}, err
	}

	return nil, errors.New("StmtQueryContext not implemented")
}

type RecordingStmtWithExecQueryContext struct {
	*RecordingStmt
	*RecordingStmtWithExecContext
	*RecordingStmtWithQueryContext
}

type RecordingRows struct {
	Rows       driver.Rows
	recorder   *apitest.Recorder
	sourceName string
	RowsFound  int
}

func (rows *RecordingRows) Columns() []string { return rows.Rows.Columns() }
func (rows *RecordingRows) Close() error {
	if rows.recorder != nil {
		rows.recorder.AddMessageResponse(apitest.MessageResponse{
			Source:    rows.sourceName,
			Target:    apitest.SystemUnderTestDefaultName,
			Header:    "SQL Result",
			Body:      fmt.Sprintf("Rows returned: %d", rows.RowsFound),
			Timestamp: time.Now().UTC(),
		})
	}

	return rows.Rows.Close()
}

func (rows *RecordingRows) Next(dest []driver.Value) error {
	err := rows.Rows.Next(dest)
	if err != io.EOF {
		rows.RowsFound++
	}

	return err
}

// see https://golang.org/src/database/sql/ctxutil.go
func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

// sqlDriverNameToDriver opens a dummy connection to get a driver
func sqlDriverNameToDriver(driverName string) driver.Driver {
	db, _ := sql.Open(driverName, "")
	if db != nil {
		db.Close()
		return db.Driver()
	}

	return nil
}
