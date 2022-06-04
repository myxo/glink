package glink

import (
	"database/sql"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type OwnInfo struct {
	Name string
	Cid  string
}

type Db struct {
	db       *sql.DB
	own_info OwnInfo
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
  CREATE TABLE IF NOT EXISTS chats (
    cid       TEXT PRIMARY KEY,
    name      TEXT,
    endpoints TEXT
  );
  CREATE TABLE IF NOT EXISTS messages (
    from_cid    TEXT,
    msg_index   INTEGER,
    to_cid      TEXT,
    create_time INTEGER,
    msg         TEXT,
    PRIMARY KEY(from_cid, msg_index)
  );
  `

	_, err = db.Exec(main_table_stmt)
	if err != nil {
		return nil, err
	}

	own_info, _ := extructOwnInfo(db)

	res := &Db{db: db, own_info: own_info}

	if res.own_info.Cid == "" {
		err = res.SetOwnCid(uuid.New().String())
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

func (d *Db) GetOwnInfo() OwnInfo {
	return d.own_info
}

func (d *Db) SetOwnName(name string) error {
	d.own_info.Name = name
	return d.doQuery(`INSERT OR REPLACE INTO chats (cid, name) VALUES (?, ?)`,
		d.own_info.Cid, d.own_info.Name)
}

func (d *Db) SetOwnCid(cid string) error {
	d.own_info.Cid = cid

	return d.doQuery(`INSERT INTO main (own_cid, timezone, db_version) VALUES (?, -3, 1)`, d.own_info.Cid)
}

func (d *Db) doQuery(query string, params ...any) error {
	trashSQL, err := d.db.Prepare(query)
	if err != nil {
		return err
	}
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

func extructOwnInfo(db *sql.DB) (OwnInfo, error) {
	var own_info OwnInfo
	rows, err := db.Query("SELECT own_cid FROM main")
	if err != nil {
		return own_info, err
	}
	rows.Next()
	err = rows.Scan(&own_info.Cid)
	if err != nil {
		return own_info, err
	}

	stmt, err := db.Prepare("SELECT name FROM chats WHERE cid = ?")
	if err != nil {
		return own_info, err
	}

	err = stmt.QueryRow(own_info.Cid).Scan(&own_info.Name)
	if err != nil {
		return own_info, err
	}

	return own_info, nil
}

func (d *Db) GetNameByCid(cid string) (string, error) {
	stmt, err := d.db.Prepare("SELECT name FROM chats WHERE cid = ?")
	if err != nil {
		return "", err
	}

	var name string
	err = stmt.QueryRow(cid).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
