package glink

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type UserLightInfo struct {
	Name string
	Uid  string
}

type Db struct {
	db       *sql.DB
	own_info UserLightInfo
}

func NewDb(path string) (*Db, error) {
	if path == "" {
		path = ":memory:"
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	main_table_stmt := `
		CREATE TABLE IF NOT EXISTS main (
		  own_cid     TEXT PRIMARY KEY, 
		  timezone    INTEGER,
		  db_version  INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS user (
		  uid             TEXT PRIMARY KEY,
		  name            TEXT,
		  endpoints       TEXT
		);
		CREATE TABLE IF NOT EXISTS chat (
		  cid             TEXT PRIMARY KEY,
		  uids            TEXT,
		  last_event_time INTEGER
		);
		CREATE TABLE IF NOT EXISTS message (
		  uid         TEXT,
		  msg_index   INTEGER,
		  cid         TEXT,
		  create_time INTEGER,
		  msg         TEXT,
		  PRIMARY KEY(uid, cid, msg_index)
		);
		`

	_, err = db.Exec(main_table_stmt)
	if err != nil {
		return nil, err
	}

	own_info, _ := extructOwnInfo(db)

	res := &Db{db: db, own_info: own_info}

	return res, nil
}

func (d *Db) GetOwnInfo() UserLightInfo {
	return d.own_info
}

func (d *Db) SetOwnName(name string) error {
	if d.own_info.Uid == "" {
		return errors.New("Cannot save name if own uid is not set")
	}
	d.own_info.Name = name
	return d.doQuery(`INSERT OR REPLACE INTO user (uid, name) VALUES (?, ?)`,
		d.own_info.Uid, d.own_info.Name)
}

func (d *Db) SetOwnCid(cid string) error {
	d.own_info.Uid = cid

	return d.doQuery(`INSERT INTO main (own_cid, timezone, db_version) 
    VALUES (?, -3, 1)`, d.own_info.Uid)
}

func (d *Db) SaveNewChat(cid string, participants []string) error {
	return d.doQuery(`INSERT INTO chat (cid, uids, last_event_time) 
    VALUES(?, ?, ?)`, cid, strings.Join(participants, ","), time.Now().UnixMicro())
}

func (d *Db) AddParticipantToChat(cid string, participant string) error {
	row, err := d.doSelect(`SELECT uids FROM chat WHERE cid = ?`, cid)
	if err != nil {
		return err
	}
	row.Next()
	var cur_participats string
	err = row.Scan(&cur_participats)
	if err != nil {
		return err
	}
	return d.doQuery(`INSERT INTO chat (cid, uids, last_event_time) 
    VALUES(?, ?, ?)`, cid, cur_participats+","+participant, time.Now().UnixMicro())
}

func (d *Db) SaveNewUid(uid string, name string, endpoints string) error {
	return d.doQuery(`INSERT INTO user (uid, name, endpoints) 
    VALUES(?, ?, ?)`, uid, name, endpoints)
}

func (d *Db) IsKnownUid(uid string) bool {
	rows, err := d.doSelect("SELECT uid FROM user WHERE uid = ?", uid)
	if err != nil {
		return false
	}
	has_value := rows.Next()
	rows.Close()
	return has_value
}

func (d *Db) SaveMessage(msg ChatMessage) error {
	err := d.doQuery(`INSERT INTO message (uid, msg_index, cid, msg)
      VALUES(?, ?, ?, ?)`, msg.FromUid, msg.Index, msg.ToCid, msg.Text)
	if err != nil {
		return err
	}
	return d.doQuery(`UPDATE chat SET last_event_time = ? WHERE cid = ?`, time.Now().UnixMicro(), msg.FromUid)
}

func (d *Db) GetMessages(cid string, from_index, to_index uint32) ([]ChatMessage, error) {
	if cid == "" {
		return nil, errors.New("cannot have empty cid")
	}
	stmt, err := d.db.Prepare(`SELECT uid, msg_index, cid, msg FROM message 
      WHERE cid = ? AND msg_index >= ? AND msg_index <= ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(cid, from_index, to_index)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]ChatMessage, 0, 10)

	for rows.Next() {
		var msg ChatMessage
		err = rows.Scan(&msg.FromUid, &msg.Index, &msg.ToCid, &msg.Text)
		if err != nil {
			return nil, err
		}
		res = append(res, msg)
	}
	return res, nil
}

func (d *Db) GetLastChats() ([]string, error) {
	stmt, err := d.db.Prepare(`SELECT cid FROM chat ORDER BY last_event_time`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]string, 0, 10)

	for rows.Next() {
		var info string
		err = rows.Scan(&info)
		if err != nil {
			return nil, err
		}
		res = append(res, info)
	}
	return res, nil
}

func (d *Db) doQuery(query string, params ...any) error {
	trashSQL, err := d.db.Prepare(query)
	if err != nil {
		return err
	}
	defer trashSQL.Close()
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Stmt(trashSQL).Exec(params...)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *Db) doSelect(query string, params ...any) (*sql.Rows, error) {
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	// TODO: cannot close here?
	defer stmt.Close()

	rows, err := stmt.Query(params...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func extructOwnInfo(db *sql.DB) (UserLightInfo, error) {
	var own_info UserLightInfo
	rows, err := db.Query("SELECT own_cid FROM main")
	if err != nil {
		return own_info, err
	}
	defer rows.Close()
	rows.Next()
	err = rows.Scan(&own_info.Uid)
	if err != nil {
		return own_info, err
	}

	stmt, err := db.Prepare("SELECT name FROM user WHERE uid = ?")
	if err != nil {
		return own_info, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(own_info.Uid).Scan(&own_info.Name)
	if err != nil {
		return own_info, err
	}

	return own_info, nil
}

func (d *Db) GetNameByCid(uid string) (string, error) {
	stmt, err := d.db.Prepare("SELECT name FROM user WHERE uid = ?")
	if err != nil {
		return "", err
	}

	var name string
	err = stmt.QueryRow(uid).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (d *Db) GetLastIndex(cid string) (uint32, error) {
	// TODO: transaction

	rows, err := d.doSelect("SELECT MAX(msg_index) FROM message WHERE cid = ?", cid)
	if err != nil {
		return 0, err
	}
	var res uint32
	rows.Next()
	err = rows.Scan(&res)
	rows.Close()
	return res, err
}
