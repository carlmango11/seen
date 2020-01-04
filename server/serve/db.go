package serve

import (
	"database/sql"
	"log"

	uuid "github.com/satori/go.uuid"
)

type DB struct {
	db *sql.DB
}

func NewDb(db *sql.DB) *DB {
	return &DB{
		db: db,
	}
}

func (d *DB) addReq(id uuid.UUID) error {
	_, err := d.db.Exec("INSERT INTO jobs (id, status, ip) VALUES (uuid_to_bin(?), 1, '192.wuuut')", id)
	if err != nil {
		log.Println("db err", err)
		return err
	}

	return nil
}

func (d *DB) status(id uuid.UUID) (Status, error) {
	var status Status
	err := d.db.QueryRow("SELECT s.name FROM jobs r INNER JOIN status s ON s.id = r.status_id WHERE r.ref = ?", id.String()).Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}

func (d *DB) setStatus(id uuid.UUID, status Status) {
	_, err := d.db.Exec("UPDATE jobs SET status = (SELECT id FROM status WHERE name = ?) WHERE id = ?", status, id)
	if err != nil {
		log.Printf("error setting job status (%d): %v", id, err)
	}
}

func (d *DB) errStatus(id uuid.UUID, jobErr error) {
	_, err := d.db.Exec("UPDATE jobs SET error = ? WHERE id = ?", jobErr.Error(), id)
	if err != nil {
		log.Printf("error writing error on job (%d): %v", id, err)
		return
	}

	d.setStatus(id, StatusError)
}
