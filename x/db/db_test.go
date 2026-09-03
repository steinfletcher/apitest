package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/steinfletcher/apitest"
)

// --- fake drivers -----------------------------------------------------------

type fakeRows struct {
	columns []string
	rows    [][]driver.Value
	pos     int
}

func (r *fakeRows) Columns() []string { return r.columns }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}

type fakeResult struct{ affected int64 }

func (r fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeResult) RowsAffected() (int64, error) { return r.affected, nil }

type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

func newRows() *fakeRows {
	return &fakeRows{columns: []string{"id"}, rows: [][]driver.Value{{int64(1)}, {int64(2)}}}
}

// legacyConn only implements the pre Go 1.8 driver interfaces.
type legacyConn struct {
	queries []string
}

func (c *legacyConn) Prepare(query string) (driver.Stmt, error) {
	return &legacyStmt{conn: c, query: query}, nil
}
func (c *legacyConn) Close() error              { return nil }
func (c *legacyConn) Begin() (driver.Tx, error) { return fakeTx{}, nil }
func (c *legacyConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	c.queries = append(c.queries, query)
	return newRows(), nil
}
func (c *legacyConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	c.queries = append(c.queries, query)
	return fakeResult{affected: 3}, nil
}

type legacyStmt struct {
	conn  *legacyConn
	query string
}

func (s *legacyStmt) Close() error  { return nil }
func (s *legacyStmt) NumInput() int { return -1 }
func (s *legacyStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.conn.Exec(s.query, args)
}
func (s *legacyStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.Query(s.query, args)
}

// contextConn implements the context aware driver interfaces.
type contextConn struct {
	queries []string
	pinged  bool
	txOpts  *driver.TxOptions
}

func (c *contextConn) Prepare(query string) (driver.Stmt, error) {
	return &contextStmt{conn: c, query: query}, nil
}
func (c *contextConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return &contextStmt{conn: c, query: query}, nil
}
func (c *contextConn) Close() error              { return nil }
func (c *contextConn) Begin() (driver.Tx, error) { return fakeTx{}, nil }
func (c *contextConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.txOpts = &opts
	return fakeTx{}, nil
}
func (c *contextConn) Ping(ctx context.Context) error {
	c.pinged = true
	return nil
}
func (c *contextConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.queries = append(c.queries, query)
	return newRows(), nil
}
func (c *contextConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.queries = append(c.queries, query)
	return fakeResult{affected: 3}, nil
}

type contextStmt struct {
	conn  *contextConn
	query string
}

func (s *contextStmt) Close() error  { return nil }
func (s *contextStmt) NumInput() int { return -1 }
func (s *contextStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, errors.New("Exec should not be used when ExecContext is available")
}
func (s *contextStmt) Query(args []driver.Value) (driver.Rows, error) {
	return nil, errors.New("Query should not be used when QueryContext is available")
}
func (s *contextStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.conn.ExecContext(ctx, s.query, args)
}
func (s *contextStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.conn.QueryContext(ctx, s.query, args)
}

type fakeDriver struct {
	conn driver.Conn
}

func (d *fakeDriver) Open(name string) (driver.Conn, error) { return d.conn, nil }

type fakeConnector struct {
	driver *fakeDriver
}

func (c *fakeConnector) Connect(context.Context) (driver.Conn, error) { return c.driver.conn, nil }
func (c *fakeConnector) Driver() driver.Driver                        { return c.driver }

// --- helpers ------------------------------------------------------------------

type message struct {
	source, target, header, body string
}

func messages(t *testing.T, recorder *apitest.Recorder) []message {
	t.Helper()
	var out []message
	for _, event := range recorder.Events {
		switch v := event.(type) {
		case apitest.MessageRequest:
			out = append(out, message{v.Source, v.Target, v.Header, v.Body})
		case apitest.MessageResponse:
			out = append(out, message{v.Source, v.Target, v.Header, v.Body})
		default:
			t.Fatalf("unexpected event type %T", event)
		}
	}
	return out
}

func assertMessages(t *testing.T, recorder *apitest.Recorder, expected ...message) {
	t.Helper()
	actual := messages(t, recorder)
	if len(actual) != len(expected) {
		t.Fatalf("expected %d messages, got %d: %+v", len(expected), len(actual), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("message %d: expected %+v, got %+v", i, expected[i], actual[i])
		}
	}
}

func drainRows(t *testing.T, rows *sql.Rows) int {
	t.Helper()
	count := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	return count
}

func openRecorded(t *testing.T, name string, conn driver.Conn, recorder *apitest.Recorder) *sql.DB {
	t.Helper()
	sql.Register(name, &fakeDriver{conn: conn})
	sql.Register(name+"-recorded", WrapWithRecorder(name, recorder))
	db, err := sql.Open(name+"-recorded", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

const (
	sut    = apitest.SystemUnderTestDefaultName
	source = "testdb"
)

// --- tests ----------------------------------------------------------------------

func TestWrapWithRecorder_RecordsContextQueriesAndExecs(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &contextConn{}
	db := openRecorded(t, source+"-ctx", conn, recorder)

	rows, err := db.QueryContext(context.Background(), "SELECT id FROM users WHERE id > ?", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := drainRows(t, rows); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}

	if _, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id = ?", 7); err != nil {
		t.Fatal(err)
	}

	assertMessages(t, recorder,
		message{sut, source + "-ctx", "SQL Query", "SELECT id FROM users WHERE id > ? [0]"},
		message{source + "-ctx", sut, "SQL Result", "Rows returned: 2"},
		message{sut, source + "-ctx", "SQL Query", "DELETE FROM users WHERE id = ? [7]"},
		message{source + "-ctx", sut, "SQL Result", "Affected rows: 3"},
	)
}

func TestWrapWithRecorder_RecordsLegacyQueriesAndExecs(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &legacyConn{}
	db := openRecorded(t, source+"-legacy", conn, recorder)

	rows, err := db.Query("SELECT id FROM users WHERE id > ?", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := drainRows(t, rows); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}

	if _, err := db.Exec("DELETE FROM users"); err != nil {
		t.Fatal(err)
	}

	assertMessages(t, recorder,
		message{sut, source + "-legacy", "SQL Query", "SELECT id FROM users WHERE id > ? [0]"},
		message{source + "-legacy", sut, "SQL Result", "Rows returned: 2"},
		message{sut, source + "-legacy", "SQL Query", "DELETE FROM users"},
		message{source + "-legacy", sut, "SQL Result", "Affected rows: 3"},
	)
}

func TestWrapWithRecorder_RecordsPreparedStatements(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &contextConn{}
	db := openRecorded(t, source+"-prepared", conn, recorder)

	stmt, err := db.Prepare("UPDATE users SET name = ? WHERE id = ?")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stmt.Close() }()

	if _, err := stmt.Exec("jan", 1); err != nil {
		t.Fatal(err)
	}

	query, err := db.Prepare("SELECT id FROM users WHERE id > ?")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = query.Close() }()

	rows, err := query.Query(0)
	if err != nil {
		t.Fatal(err)
	}
	drainRows(t, rows)

	assertMessages(t, recorder,
		message{sut, source + "-prepared", "SQL Query", "UPDATE users SET name = ? WHERE id = ? [jan 1]"},
		message{source + "-prepared", sut, "SQL Result", "Affected rows: 3"},
		message{sut, source + "-prepared", "SQL Query", "SELECT id FROM users WHERE id > ? [0]"},
		message{source + "-prepared", sut, "SQL Result", "Rows returned: 2"},
	)
}

func TestWrapWithRecorder_RecordsLegacyPreparedStatements(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &legacyConn{}
	db := openRecorded(t, source+"-legacy-prepared", conn, recorder)

	stmt, err := db.Prepare("UPDATE users SET name = ? WHERE id = ?")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stmt.Close() }()

	if _, err := stmt.Exec("jan", 1); err != nil {
		t.Fatal(err)
	}

	assertMessages(t, recorder,
		message{sut, source + "-legacy-prepared", "SQL Query", "UPDATE users SET name = ? WHERE id = ? [jan 1]"},
		message{source + "-legacy-prepared", sut, "SQL Result", "Affected rows: 3"},
	)
}

func TestWrapWithRecorder_PassesThroughPingAndTransactions(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &contextConn{}
	db := openRecorded(t, source+"-tx", conn, recorder)

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !conn.pinged {
		t.Fatal("expected Ping to reach the underlying connection")
	}

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if conn.txOpts == nil || !conn.txOpts.ReadOnly {
		t.Fatalf("expected BeginTx options to reach the underlying connection, got %+v", conn.txOpts)
	}

	assertMessages(t, recorder)
}

func TestWrapWithRecorder_ToleratesANilRecorder(t *testing.T) {
	conn := &contextConn{}
	db := openRecorded(t, source+"-nil-recorder", conn, nil)

	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		t.Fatal(err)
	}
	if got := drainRows(t, rows); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}
	if _, err := db.Exec("DELETE FROM users"); err != nil {
		t.Fatal(err)
	}
}

func TestWrapConnectorWithRecorder_RecordsQueries(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &contextConn{}
	connector := WrapConnectorWithRecorder(&fakeConnector{driver: &fakeDriver{conn: conn}}, source, recorder)
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		t.Fatal(err)
	}
	drainRows(t, rows)

	assertMessages(t, recorder,
		message{sut, source, "SQL Query", "SELECT id FROM users"},
		message{source, sut, "SQL Result", "Rows returned: 2"},
	)
}

func TestWrapWithRecorder_RecordedRowsCountEveryRowRead(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	rows := &recordingRows{Rows: newRows(), recorder: recorder, sourceName: source}

	dest := make([]driver.Value, 1)
	for {
		if err := rows.Next(dest); err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			break
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	assertMessages(t, recorder, message{source, sut, "SQL Result", "Rows returned: 2"})
}

func TestNamedValueToValue_RejectsNamedParameters(t *testing.T) {
	_, err := namedValueToValue([]driver.NamedValue{{Name: "id", Value: 1}})
	if err == nil {
		t.Fatal("expected named parameters to be rejected")
	}

	values, err := namedValueToValue([]driver.NamedValue{{Ordinal: 1, Value: 1}, {Ordinal: 2, Value: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != 1 || values[1] != "a" {
		t.Fatalf("unexpected values %v", values)
	}
}

func TestSQLDriverNameToDriver_ReturnsNilForUnknownDrivers(t *testing.T) {
	if got := sqlDriverNameToDriver("no-such-driver"); got != nil {
		t.Fatalf("expected nil, got %T", got)
	}
}

// --- fakes for error paths and interface combinations ---------------------------

var errFake = errors.New("fake failure")

// plainConn implements only driver.Conn.
type plainConn struct{}

func (plainConn) Prepare(query string) (driver.Stmt, error) { return &plainStmt{}, nil }
func (plainConn) Close() error                              { return nil }
func (plainConn) Begin() (driver.Tx, error)                 { return fakeTx{}, nil }

type plainStmt struct{}

func (plainStmt) Close() error  { return nil }
func (plainStmt) NumInput() int { return -1 }
func (plainStmt) Exec(args []driver.Value) (driver.Result, error) {
	return fakeResult{affected: 1}, nil
}
func (plainStmt) Query(args []driver.Value) (driver.Rows, error) { return newRows(), nil }

// failingConn implements every interface and fails every call.
type failingConn struct{}

func (failingConn) Prepare(string) (driver.Stmt, error) { return nil, errFake }
func (failingConn) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return nil, errFake
}
func (failingConn) Close() error                                      { return nil }
func (failingConn) Begin() (driver.Tx, error)                         { return nil, errFake }
func (failingConn) Query(string, []driver.Value) (driver.Rows, error) { return nil, errFake }
func (failingConn) Exec(string, []driver.Value) (driver.Result, error) {
	return nil, errFake
}
func (failingConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, errFake
}
func (failingConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, errFake
}

type failingStmt struct{}

func (failingStmt) Close() error                               { return nil }
func (failingStmt) NumInput() int                              { return -1 }
func (failingStmt) Exec([]driver.Value) (driver.Result, error) { return nil, errFake }
func (failingStmt) Query([]driver.Value) (driver.Rows, error)  { return nil, errFake }
func (failingStmt) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	return nil, errFake
}
func (failingStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return nil, errFake
}

// queryOnlyStmt and execOnlyStmt implement just one of the context statement interfaces.
type queryOnlyStmt struct{ plainStmt }

func (queryOnlyStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return newRows(), nil
}

type execOnlyStmt struct{ plainStmt }

func (execOnlyStmt) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	return fakeResult{affected: 2}, nil
}

// stmtConn returns a configurable statement from PrepareContext and Prepare.
type stmtConn struct {
	plainConn
	stmt driver.Stmt
}

func (c stmtConn) Prepare(string) (driver.Stmt, error)                         { return c.stmt, nil }
func (c stmtConn) PrepareContext(context.Context, string) (driver.Stmt, error) { return c.stmt, nil }

type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) { return nil, errFake }

type contextDriver struct {
	conn driver.Conn
	fail bool
}

func (d *contextDriver) Open(string) (driver.Conn, error) { return d.conn, nil }
func (d *contextDriver) OpenConnector(string) (driver.Connector, error) {
	if d.fail {
		return nil, errFake
	}
	return &contextConnector{driver: d}, nil
}

type contextConnector struct {
	driver *contextDriver
}

func (c *contextConnector) Connect(context.Context) (driver.Conn, error) { return c.driver.conn, nil }
func (c *contextConnector) Driver() driver.Driver                        { return c.driver }

type failingConnector struct{}

func (failingConnector) Connect(context.Context) (driver.Conn, error) { return nil, errFake }
func (failingConnector) Driver() driver.Driver                        { return failingDriver{} }

func assertErr(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || err.Error() != expected {
		t.Fatalf("expected error %q, got %v", expected, err)
	}
}

// --- tests -----------------------------------------------------------------------

func TestWrapWithRecorder_WrapsDriverContextDrivers(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	sql.Register("ctx-driver", &contextDriver{conn: &contextConn{}})
	wrapped := WrapWithRecorder("ctx-driver", recorder)

	if _, ok := wrapped.(driver.DriverContext); !ok {
		t.Fatalf("expected a DriverContext wrapper, got %T", wrapped)
	}
	sql.Register("ctx-driver-recorded", wrapped)
	db, err := sql.Open("ctx-driver-recorded", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT id FROM users")
	if err != nil {
		t.Fatal(err)
	}
	drainRows(t, rows)
	assertMessages(t, recorder,
		message{sut, "ctx-driver", "SQL Query", "SELECT id FROM users"},
		message{"ctx-driver", sut, "SQL Result", "Rows returned: 2"},
	)

	if _, err := (&recordingDriverContext{&recordingDriver{Driver: &contextDriver{fail: true}}}).OpenConnector(""); !errors.Is(err, errFake) {
		t.Fatalf("expected the connector error, got %v", err)
	}
	_, err = (&recordingDriverContext{&recordingDriver{Driver: &fakeDriver{}}}).OpenConnector("")
	assertErr(t, err, "OpenConnector not implemented")
}

func TestRecordingDriver_OpenVariants(t *testing.T) {
	if _, err := (&recordingDriver{Driver: failingDriver{}}).Open(""); !errors.Is(err, errFake) {
		t.Fatalf("expected the open error, got %v", err)
	}

	conn, err := (&recordingDriver{Driver: &fakeDriver{conn: plainConn{}}}).Open("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := conn.(*recordingConn); !ok {
		t.Fatalf("expected a plain recording conn, got %T", conn)
	}
}

func TestRecordingConnector_ConnectVariants(t *testing.T) {
	recorder := apitest.NewTestRecorder()

	if _, err := (&recordingConnector{Connector: failingConnector{}}).Connect(context.Background()); !errors.Is(err, errFake) {
		t.Fatalf("expected the connect error, got %v", err)
	}

	legacy, err := (&recordingConnector{recorder: recorder, sourceName: source, Connector: &fakeConnector{driver: &fakeDriver{conn: &legacyConn{}}}}).Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := legacy.(*recordingConnWithExecQuery); !ok {
		t.Fatalf("expected a legacy recording conn, got %T", legacy)
	}

	plain, err := (&recordingConnector{Connector: &fakeConnector{driver: &fakeDriver{conn: plainConn{}}}}).Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := plain.(*recordingConn); !ok {
		t.Fatalf("expected a plain recording conn, got %T", plain)
	}

	connector := &recordingConnector{Connector: &fakeConnector{driver: &fakeDriver{}}}
	if _, ok := connector.Driver().(*fakeDriver); !ok {
		t.Fatalf("expected the underlying driver, got %T", connector.Driver())
	}
}

func TestRecordingConn_LegacyPaths(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &legacyConn{}
	db := openRecorded(t, source+"-legacy-paths", conn, recorder)

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec("DELETE FROM users WHERE id = ?", 7); err != nil {
		t.Fatal(err)
	}

	stmt, err := db.Prepare("SELECT id FROM users WHERE id > ?")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stmt.Close() }()
	rows, err := stmt.Query(0)
	if err != nil {
		t.Fatal(err)
	}
	drainRows(t, rows)

	assertMessages(t, recorder,
		message{sut, source + "-legacy-paths", "SQL Query", "DELETE FROM users WHERE id = ? [7]"},
		message{source + "-legacy-paths", sut, "SQL Result", "Affected rows: 3"},
		message{sut, source + "-legacy-paths", "SQL Query", "SELECT id FROM users WHERE id > ? [0]"},
		message{source + "-legacy-paths", sut, "SQL Result", "Rows returned: 2"},
	)
}

func TestRecordingConn_PrepareReturnsContextAwareStatements(t *testing.T) {
	recorder := apitest.NewTestRecorder()
	conn := &recordingConn{Conn: stmtConn{stmt: &contextStmt{conn: &contextConn{}}}, recorder: recorder, sourceName: source}

	stmt, err := conn.Prepare("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stmt.(*recordingStmtWithExecQueryContext); !ok {
		t.Fatalf("expected a context aware statement, got %T", stmt)
	}

	if _, err := (&recordingConn{Conn: failingConn{}}).Prepare("SELECT 1"); !errors.Is(err, errFake) {
		t.Fatalf("expected the prepare error, got %v", err)
	}
}

func TestRecordingConn_PrepareContextVariants(t *testing.T) {
	for name, test := range map[string]struct {
		stmt     driver.Stmt
		expected string
	}{
		"query only": {queryOnlyStmt{}, "*db.recordingStmtWithQueryContext"},
		"exec only":  {execOnlyStmt{}, "*db.recordingStmtWithExecContext"},
		"neither":    {plainStmt{}, "*db.recordingStmt"},
	} {
		t.Run(name, func(t *testing.T) {
			conn := &recordingConnWithPrepareContext{&recordingConn{Conn: stmtConn{stmt: test.stmt}}}
			stmt, err := conn.PrepareContext(context.Background(), "SELECT 1")
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%T", stmt); got != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, got)
			}
		})
	}

	if _, err := (&recordingConnWithPrepareContext{&recordingConn{Conn: failingConn{}}}).PrepareContext(context.Background(), "SELECT 1"); !errors.Is(err, errFake) {
		t.Fatalf("expected the prepare error, got %v", err)
	}
	_, err := (&recordingConnWithPrepareContext{&recordingConn{Conn: plainConn{}}}).PrepareContext(context.Background(), "SELECT 1")
	assertErr(t, err, "ConnPrepareContext not implemented")
}

func TestRecordingConn_UnderlyingErrorsArePropagated(t *testing.T) {
	base := &recordingConn{Conn: failingConn{}, recorder: apitest.NewTestRecorder(), sourceName: source}
	ctx := context.Background()

	if _, err := (&recordingConnWithQuery{base}).Query("q", nil); !errors.Is(err, errFake) {
		t.Fatalf("Query: %v", err)
	}
	if _, err := (&recordingConnWithExec{base}).Exec("q", nil); !errors.Is(err, errFake) {
		t.Fatalf("Exec: %v", err)
	}
	if _, err := (&recordingConnWithQueryContext{base}).QueryContext(ctx, "q", nil); !errors.Is(err, errFake) {
		t.Fatalf("QueryContext: %v", err)
	}
	if _, err := (&recordingConnWithExecContext{base}).ExecContext(ctx, "q", nil); !errors.Is(err, errFake) {
		t.Fatalf("ExecContext: %v", err)
	}
	if len(base.recorder.Events) != 0 {
		t.Fatalf("expected nothing to be recorded for failed calls, got %d events", len(base.recorder.Events))
	}
}

func TestRecordingConn_NotImplementedFallbacks(t *testing.T) {
	base := &recordingConn{Conn: plainConn{}}
	ctx := context.Background()

	_, err := (&recordingConnWithQuery{base}).Query("q", nil)
	assertErr(t, err, "Queryer not implemented")
	_, err = (&recordingConnWithExec{base}).Exec("q", nil)
	assertErr(t, err, "Execer not implemented")
	_, err = (&recordingConnWithQueryContext{base}).QueryContext(ctx, "q", nil)
	assertErr(t, err, "QueryerContext not implemented")
	_, err = (&recordingConnWithExecContext{base}).ExecContext(ctx, "q", nil)
	assertErr(t, err, "ExecerContext not implemented")
	_, err = (&recordingConnWithBeginTx{base}).BeginTx(ctx, driver.TxOptions{})
	assertErr(t, err, "ConnBeginTx not implemented")
	assertErr(t, (&recordingConnWithPing{base}).Ping(ctx), "Pinger not implemented")
}

func TestRecordingConn_NamedParametersAreRejected(t *testing.T) {
	base := &recordingConn{Conn: &contextConn{}, recorder: apitest.NewTestRecorder(), sourceName: source}
	named := []driver.NamedValue{{Name: "id", Value: 1}}
	ctx := context.Background()
	const expected = "sql: driver does not support the use of Named Parameters"

	_, err := (&recordingConnWithQueryContext{base}).QueryContext(ctx, "q", named)
	assertErr(t, err, expected)
	_, err = (&recordingConnWithExecContext{base}).ExecContext(ctx, "q", named)
	assertErr(t, err, expected)

	stmt := &recordingStmt{Stmt: &contextStmt{conn: &contextConn{}}, recorder: base.recorder, sourceName: source, query: "q"}
	_, err = (&recordingStmtWithQueryContext{stmt}).QueryContext(ctx, named)
	assertErr(t, err, expected)
	_, err = (&recordingStmtWithExecContext{stmt}).ExecContext(ctx, named)
	assertErr(t, err, expected)
}

func TestRecordingStmt_ErrorsAndFallbacks(t *testing.T) {
	ctx := context.Background()
	failing := &recordingStmt{Stmt: failingStmt{}, recorder: apitest.NewTestRecorder(), sourceName: source, query: "q"}

	if _, err := (&recordingStmtWithQueryContext{failing}).QueryContext(ctx, nil); !errors.Is(err, errFake) {
		t.Fatalf("QueryContext: %v", err)
	}
	if _, err := (&recordingStmtWithExecContext{failing}).ExecContext(ctx, nil); !errors.Is(err, errFake) {
		t.Fatalf("ExecContext: %v", err)
	}
	if _, err := failing.Exec(nil); !errors.Is(err, errFake) {
		t.Fatalf("Exec: %v", err)
	}
	if _, err := failing.Query(nil); !errors.Is(err, errFake) {
		t.Fatalf("Query: %v", err)
	}

	plain := &recordingStmt{Stmt: plainStmt{}}
	_, err := (&recordingStmtWithQueryContext{plain}).QueryContext(ctx, nil)
	assertErr(t, err, "StmtQueryContext not implemented")
	_, err = (&recordingStmtWithExecContext{plain}).ExecContext(ctx, nil)
	assertErr(t, err, "StmtExecContext not implemented")
}
