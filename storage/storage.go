package storage

import (
	"database/sql"
	"fmt"
	"log"
	"tgbot/models"
	"time"

	_ "github.com/lib/pq"
)

func OpenDatabase(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func AddUserToDatabase(db *sql.DB, userID int, userName string) error {
	query := `INSERT INTO users (user_id, name) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`
	_, err := db.Exec(query, userID, userName)
	return err
}

func AddChannelToDatabase(db *sql.DB, channelLink string) error {
	query := `INSERT INTO channels (name) VALUES ($1)`
	_, err := db.Exec(query, channelLink)
	return err
}

func AddFileMetadataToDatabase(db *sql.DB, fileID, fileName, mimeType string, fileData []byte) error {
	// Truncate the table
	_, err := db.Exec(`TRUNCATE TABLE files`)
	if err != nil {
		return fmt.Errorf("error truncating files table: %v", err)
	}

	// Insert new file metadata
	query := `INSERT INTO files (file_id, file_name, mime_type, file_data) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(query, fileID, fileName, mimeType, fileData)
	if err != nil {
		return fmt.Errorf("error inserting file metadata: %v", err)
	}

	return nil
}

func AddAnswerToDatabase(db *sql.DB, answer string) error {

	_, err := db.Exec(`TRUNCATE TABLE answers`)
	if err != nil {
		return fmt.Errorf("error truncating files table: %v", err)
	}

	query := `INSERT INTO answers (answers) VALUES ($1)`
	_, err = db.Exec(query, answer)
	return err
}

func GetChannelsFromDatabase(db *sql.DB) ([]string, error) {
	query := `SELECT name FROM channels`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []string
	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

func GetFileFromDatabase(db *sql.DB) (fileID, fileName string, err error) {
	query := `SELECT file_id, file_name FROM files LIMIT 1`
	row := db.QueryRow(query)
	err = row.Scan(&fileID, &fileName)
	return
}

func GetCorrectAnswersFromDatabase(db *sql.DB) (string, error) {
	query := `SELECT answers FROM answers LIMIT 1`
	var answers string
	err := db.QueryRow(query).Scan(&answers)
	return answers, err
}

func TruncateAnswersTable(db *sql.DB) error {
	_, err := db.Exec(`TRUNCATE TABLE answers`)
	return err
}

func AddAnswersToDatabase(db *sql.DB, fileData []byte) error {
	answers := string(fileData)
	query := `INSERT INTO answers (answers) VALUES ($1)`
	_, err := db.Exec(query, answers)
	return err
}


func AddAdminToDatabase(db *sql.DB, adminID int64) error {
	query := `INSERT INTO admins (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(query, adminID)
	return err
}

func RemoveAdminFromDatabase(db *sql.DB, adminID int64) error {
	query := `DELETE FROM admins WHERE id = $1`
	_, err := db.Exec(query, adminID)
	return err
}


func IsAdmin(userID int, db *sql.DB) bool {
	var id int
	query := `SELECT id FROM admins WHERE id = $1`
	err := db.QueryRow(query, userID).Scan(&id)
	return err == nil
}

func DeleteChannelFromDatabase(db *sql.DB, channel string) error {
	query := `DELETE FROM channels WHERE name = $1`
	_, err := db.Exec(query, channel)
	return err
}

func GetTotalUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func GetTodayUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= $1", time.Now().Truncate(24*time.Hour)).Scan(&count)
	return count, err
}

func GetLastMonthUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= $1", time.Now().AddDate(0, -1, 0)).Scan(&count)
	return count, err
}

func GetAllUsers(db *sql.DB) ([]models.User, error) {
	log.Println("GetAllUsers funksiyasi ishga tushdi") // Log qo'shish
	query := `SELECT user_id, name, status FROM users`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Status); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}