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
	recordingDriver := &recordingDriver{
		sourceName: driverName,
		Driver:     sqlDriver,
		recorder:   recorder,
	}

	if _, ok := sqlDriver.(driver.DriverContext); ok {
		return &recordingDriverContext{recordingDriver}
	}

	return recordingDriver
}

// WrapConnectorWithRecorder wraps an existing connector with a Recorder
func WrapConnectorWithRecorder(connector driver.Connector, sourceName string, recorder *apitest.Recorder) driver.Connector {
	return &recordingConnector{recorder: recorder, sourceName: sourceName, Connector: connector}
}

type recordingDriver struct {
	Driver     driver.Driver
	recorder   *apitest.Recorder
	sourceName string
}

// Open wraps opening a connection to the database with a recorder
func (d *recordingDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	_, isConnQuery := conn.(driver.Queryer)
	_, isConnQueryCtx := conn.(driver.QueryerContext)
	_, isConnExec := conn.(driver.Execer)
	_, isConnExecCtx := conn.(driver.ExecerContext)
	_, isConnPrepareCtx := conn.(driver.ConnPrepareContext)
	recordingConn := &recordingConn{Conn: conn, recorder: d.recorder, sourceName: d.sourceName}

	if isConnQueryCtx && isConnExecCtx && isConnPrepareCtx {
		return &recordingConnWithExecQueryPrepareContext{
			recordingConn,
			&recordingConnWithPrepareContext{recordingConn},
			&recordingConnWithExecContext{recordingConn},
			&recordingConnWithQueryContext{recordingConn},
			&recordingConnWithBeginTx{recordingConn},
			&recordingConnWithPing{recordingConn},
		}, nil
	}

	if isConnQuery && isConnExec {
		return &recordingConnWithExecQuery{
			recordingConn,
			&recordingConnWithExec{recordingConn},
			&recordingConnWithQuery{recordingConn},
		}, nil
	}

	return recordingConn, nil
}

type recordingDriverContext struct {
	*recordingDriver
}

// OpenConnector wraps the underlying driver OpenConnector call
func (d *recordingDriverContext) OpenConnector(name string) (driver.Connector, error) {
	if driverCtx, ok := d.Driver.(driver.DriverContext); ok {
		connector, err := driverCtx.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &recordingConnector{recorder: d.recorder, sourceName: d.sourceName, Connector: connector}, nil
	}

	return nil, errors.New("OpenConnector not implemented")
}

type recordingConnector struct {
	Connector  driver.Connector
	recorder   *apitest.Recorder
	sourceName string
}

// Connect wraps the
func (c *recordingConnector) Connect(context context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(context)
	if err != nil {
		return nil, err
	}

	_, isConnQuery := conn.(driver.Queryer)
	_, isConnQueryCtx := conn.(driver.QueryerContext)
	_, isConnExec := conn.(driver.Execer)
	_, isConnExecCtx := conn.(driver.ExecerContext)
	_, isConnPrepareCtx := conn.(driver.ConnPrepareContext)
	recordingConn := &recordingConn{Conn: conn, recorder: c.recorder, sourceName: c.sourceName}

	if isConnQueryCtx && isConnExecCtx && isConnPrepareCtx {
		return &recordingConnWithExecQueryPrepareContext{
			recordingConn,
			&recordingConnWithPrepareContext{recordingConn},
			&recordingConnWithExecContext{recordingConn},
			&recordingConnWithQueryContext{recordingConn},
			&recordingConnWithBeginTx{recordingConn},
			&recordingConnWithPing{recordingConn},
		}, nil
	}

	if isConnQuery && isConnExec {
		return &recordingConnWithExecQuery{
			recordingConn,
			&recordingConnWithExec{recordingConn},
			&recordingConnWithQuery{recordingConn},
		}, nil
	}

	return recordingConn, nil
}

func (c *recordingConnector) Driver() driver.Driver { return c.Connector.Driver() }

type recordingConn struct {
	Conn       driver.Conn
	recorder   *apitest.Recorder
	sourceName string
}

// Prepare wraps the underlying prepare statement call with a recorder, allowing the statement to be observed
func (conn *recordingConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := conn.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}

	_, isStmtQueryContext := stmt.(driver.StmtQueryContext)
	_, isStmtExecContext := stmt.(driver.StmtExecContext)
	recordingStmt := &recordingStmt{
		Stmt:       stmt,
		recorder:   conn.recorder,
		query:      query,
		sourceName: conn.sourceName,
	}

	if isStmtQueryContext && isStmtExecContext {
		return &recordingStmtWithExecQueryContext{
			recordingStmt,
			&recordingStmtWithExecContext{recordingStmt},
			&recordingStmtWithQueryContext{recordingStmt},
		}, nil
	}

	return recordingStmt, nil
}

// Close will close the wrapped drivers connection
func (conn *recordingConn) Close() error { return conn.Conn.Close() }

//Begin begins a new transaction for the wrapped driver
func (conn *recordingConn) Begin() (driver.Tx, error) { return conn.Conn.Begin() }

type recordingConnWithQuery struct {
	*recordingConn
}

// Query executes a query with arguments for the wrapped driver, capturing the query to the recorder
func (conn *recordingConnWithQuery) Query(query string, args []driver.Value) (driver.Rows, error) {
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

		return &recordingRows{Rows: rows, recorder: conn.recorder, sourceName: conn.sourceName}, err
	}

	return nil, errors.New("Queryer not implemented")
}

type recordingConnWithQueryContext struct {
	*recordingConn
}

// QueryContext wraps the underlying drivers queryContext call, capturing to the recorder
func (conn *recordingConnWithQueryContext) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
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

		return &recordingRows{Rows: rows, recorder: conn.recorder, sourceName: conn.sourceName}, err
	}

	return nil, errors.New("QueryerContext not implemented")
}

type recordingConnWithExec struct {
	*recordingConn
}

// Exec wraps the underlying drivers Exec call, capturing to the recorder
func (conn *recordingConnWithExec) Exec(query string, args []driver.Value) (driver.Result, error) {
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

type recordingConnWithExecContext struct {
	*recordingConn
}

// ExecContext wraps the underlying drivers ExecContext call, capturing to the recorder
func (conn *recordingConnWithExecContext) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
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

type recordingConnWithPrepareContext struct {
	*recordingConn
}

// PrepareContext wraps the underlying drivers PrepareContext call, capturing to the recorder
func (conn *recordingConnWithPrepareContext) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if connPrepareCtx, ok := conn.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := connPrepareCtx.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}

		_, isStmtQueryContext := stmt.(driver.StmtQueryContext)
		_, isStmtExecContext := stmt.(driver.StmtExecContext)
		recordingStmt := &recordingStmt{Stmt: stmt, recorder: conn.recorder, query: query, sourceName: conn.sourceName}

		if isStmtQueryContext && isStmtExecContext {
			return &recordingStmtWithExecQueryContext{
				recordingStmt,
				&recordingStmtWithExecContext{recordingStmt},
				&recordingStmtWithQueryContext{recordingStmt},
			}, nil
		}

		if isStmtQueryContext {
			return &recordingStmtWithQueryContext{recordingStmt}, nil
		}

		if isStmtExecContext {
			return &recordingStmtWithExecContext{recordingStmt}, nil
		}

		return recordingStmt, nil

	}

	return nil, errors.New("ConnPrepareContext not implemented")
}

type recordingConnWithBeginTx struct {
	*recordingConn
}

// BeginTx wraps the underlying drivers BeginTx call
func (conn *recordingConnWithBeginTx) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if connBeginTx, ok := conn.Conn.(driver.ConnBeginTx); ok {
		return connBeginTx.BeginTx(ctx, opts)
	}

	return nil, errors.New("ConnBeginTx not implemented")
}

type recordingConnWithPing struct {
	*recordingConn
}

// Ping wraps the underlying drivers Ping call
func (conn *recordingConnWithPing) Ping(ctx context.Context) error {
	if connPinger, ok := conn.Conn.(driver.Pinger); ok {
		return connPinger.Ping(ctx)
	}

	return errors.New("Pinger not implemented")
}

type recordingConnWithExecQuery struct {
	*recordingConn
	*recordingConnWithExec
	*recordingConnWithQuery
}

type recordingConnWithExecQueryPrepareContext struct {
	*recordingConn
	*recordingConnWithPrepareContext
	*recordingConnWithExecContext
	*recordingConnWithQueryContext
	*recordingConnWithBeginTx
	*recordingConnWithPing
}

type recordingStmt struct {
	Stmt       driver.Stmt
	recorder   *apitest.Recorder
	sourceName string
	query      string
}

// Close wraps the underlying drivers Close call
func (stmt *recordingStmt) Close() error {
	return stmt.Stmt.Close()
}

// NumInput wraps the underlying drivers NumInput call
func (stmt *recordingStmt) NumInput() int {
	return stmt.Stmt.NumInput()
}

// Exec wraps the underlying drivers Exec call, capturing to the recorder
func (stmt *recordingStmt) Exec(args []driver.Value) (driver.Result, error) {
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

// Query executes a query with arguments for the wrapped driver, capturing the query to the recorder
func (stmt *recordingStmt) Query(args []driver.Value) (driver.Rows, error) {
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

	return &recordingRows{Rows: rows, recorder: stmt.recorder, sourceName: stmt.sourceName}, err
}

type recordingStmtWithExecContext struct {
	*recordingStmt
}

// ExecContext wraps the underlying drivers ExecContext call, capturing to the recorder
func (stmt *recordingStmtWithExecContext) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
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

type recordingStmtWithQueryContext struct {
	*recordingStmt
}

// QueryContext wraps the underlying drivers QueryContext call, capturing to the recorder
func (stmt *recordingStmtWithQueryContext) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
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

		return &recordingRows{Rows: rows, recorder: stmt.recorder, sourceName: stmt.sourceName}, err
	}

	return nil, errors.New("StmtQueryContext not implemented")
}

type recordingStmtWithExecQueryContext struct {
	*recordingStmt
	*recordingStmtWithExecContext
	*recordingStmtWithQueryContext
}

type recordingRows struct {
	Rows       driver.Rows
	recorder   *apitest.Recorder
	sourceName string
	RowsFound  int
}

// Columns wraps the underlying drivers Columns call
func (rows *recordingRows) Columns() []string { return rows.Rows.Columns() }

// Close wraps the underlying drivers Close call, capturing to the recorder
func (rows *recordingRows) Close() error {
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

// underlying wraps the underlying drivers underlying call
func (rows *recordingRows) Next(dest []driver.Value) error {
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
