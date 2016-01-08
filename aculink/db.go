package aculink

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var columnNames = []string{
	"uuid",
	"timestamp",
	"bridge_id",
	"sensor",
	"mt",
	"battery",
	"signal_rssi",
	"temperature_c",
	"humidity",
	"wind_kmh",
	"wind_direction",
	"rainfall_mm",
	"pressure_pa",
}

type DB struct {
	*sql.DB
	uuidExistsStmt *sql.Stmt
	insertStmt     *sql.Stmt
}

func OpenDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	uuidExistsStmt, err := db.Prepare("SELECT id FROM data WHERE uuid = ?")
	if err != nil {
		db.Close()
		return nil, err
	}

	columns := strings.Join(columnNames, ",")
	args := ""
	for i, _ := range columnNames {
		if i == 0 {
			args += "?"
		} else {
			args += ", ?"
		}
	}

	insertStmt, err := db.Prepare(
		fmt.Sprintf(`INSERT INTO data(%s) VALUES(%s)`, columns, args),
	)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{
		DB:             db,
		uuidExistsStmt: uuidExistsStmt,
		insertStmt:     insertStmt,
	}, nil
}

func (self *DB) UUIDExists(uuid string) (bool, error) {
	var existing_id uint64
	err := self.uuidExistsStmt.QueryRow(uuid).Scan(&existing_id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return existing_id > 0, err
}

func (self *DB) float32PtrToSQL(f *float32) (fret sql.NullFloat64) {
	if f != nil {
		fret.Float64 = float64(*f)
		fret.Valid = true
	}
	return
}

func (self *DB) intPtrToSQL(i *int) (iret sql.NullInt64) {
	if i != nil {
		iret.Int64 = int64(*i)
		iret.Valid = true
	}
	return
}

func (self *DB) stringPtrToSQL(s *string) (sret sql.NullString) {
	if s != nil {
		sret.String = *s
		sret.Valid = true
	}
	return
}

func (self *DB) InsertData(data *Data) error {
	_, err := self.insertStmt.Exec(
		data.UUID.String(),
		data.Timestamp.Time,
		data.BridgeID,
		data.Sensor,
		data.Mt,
		self.stringPtrToSQL(data.Battery),
		self.intPtrToSQL(data.SignalRSSI),
		self.float32PtrToSQL(data.TemperatureC),
		self.float32PtrToSQL(data.Humidity),
		self.float32PtrToSQL(data.WindKMH),
		self.float32PtrToSQL(data.WindDirection),
		self.float32PtrToSQL(data.RainfallMM),
		self.intPtrToSQL(data.PressurePA),
	)

	return err
}
