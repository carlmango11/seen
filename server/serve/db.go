package serve

import (
	"database/sql"
	"errors"
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

func (d *DB) matchesIp(id string, ip string) bool {
	if len(ip) == 0 {
		log.Printf("cannot do check on blank ip: %v", id)
		return false
	}

	var originalIp string
	err := d.db.QueryRow("SELECT ip FROM jobs WHERE id = uuid_to_bin(?)", id).Scan(&originalIp)
	if err != nil {
		log.Printf("error reading ip for %v: %v", id, err)
		return false
	}

	return originalIp == ip
}

func (d *DB) writeGuideJson(id uuid.UUID, guideJson string) error {
	res, err := d.db.Exec("UPDATE jobs SET guide_json = ? WHERE id = uuid_to_bin(?)", guideJson, id.String())
	if err != nil {
		return err
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		log.Printf("%v not found when setting json", id)
		return errors.New("job not found")
	}

	return nil
}

func (d *DB) addReq(id uuid.UUID, ip string) error {
	_, err := d.db.Exec("INSERT INTO jobs (id, status_id, ip, created_at) VALUES (uuid_to_bin(?), (SELECT id FROM status WHERE name = ?), ?, now())", id, StatusIncoming, ip)
	if err != nil {
		log.Println("db err", err)
		return err
	}

	return nil
}

func (d *DB) getStatus(id uuid.UUID) (Status, error) {
	var status Status
	err := d.db.QueryRow("SELECT s.name FROM jobs r INNER JOIN status s ON s.id = r.status_id WHERE r.id = uuid_to_bin(?)", id.String()).Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}

func (d *DB) getGuides(id uuid.UUID) (string, error) {
	var guideJson string
	err := d.db.QueryRow("SELECT guide_json FROM jobs WHERE id = uuid_to_bin(?)", id).Scan(&guideJson)
	if err != nil {
		return "", err
	}

	return guideJson, nil
}

func (d *DB) setComplete(id uuid.UUID) {
	res, err := d.db.Exec("UPDATE jobs SET completed_at = now(), status_id = (SELECT id FROM status WHERE name = ?) WHERE id = uuid_to_bin(?)", StatusComplete, id)
	if err != nil {
		log.Printf("error setting job status (%d): %v", id, err)
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		log.Printf("%v not found when setting status", id)
	}
}

func (d *DB) setStatus(id uuid.UUID, status Status) {
	res, err := d.db.Exec("UPDATE jobs SET status_id = (SELECT id FROM status WHERE name = ?) WHERE id = uuid_to_bin(?)", status, id)
	if err != nil {
		log.Printf("error setting job status (%d): %v", id, err)
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		log.Printf("%v not found when setting status", id)
	}
}

func (d *DB) errStatus(id uuid.UUID, jobErr error) {
	res, err := d.db.Exec("UPDATE jobs SET error = ? WHERE id = uuid_to_bin(?)", jobErr.Error(), id)
	if err != nil {
		log.Printf("error writing error on job (%d): %v", id, err)
		return
	}

	n, _ := res.RowsAffected()
	if n != 1 {
		log.Printf("%v not found when setting error status", id)
	}

	d.setStatus(id, StatusError)
}
