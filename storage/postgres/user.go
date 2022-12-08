package postgres

import (
	"database/sql"
	"fmt"

	"github.com/TemurMannonov/medium_user_service/pkg/utils"
	"github.com/TemurMannonov/medium_user_service/storage/repo"
	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUser(db *sqlx.DB) repo.UserStorageI {
	return &userRepo{
		db: db,
	}
}

func (ur *userRepo) Create(user *repo.User) (*repo.User, error) {

	query := `
		INSERT INTO users(
			first_name,
			last_name,
			phone_number,
			email,
			gender,
			password,
			username,
			profile_image_url,
			type
		) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`

	row := ur.db.QueryRow(
		query,
		user.FirstName,
		user.LastName,
		utils.NullString(user.PhoneNumber),
		user.Email,
		utils.NullString(user.Gender),
		user.Password,
		utils.NullString(user.Username),
		utils.NullString(user.ProfileImageUrl),
		user.Type,
	)

	err := row.Scan(
		&user.ID,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (ur *userRepo) Get(id int64) (*repo.User, error) {
	var (
		result                                         repo.User
		phoneNumber, gender, username, profileImageUrl sql.NullString
	)

	query := `
		SELECT
			id,
			first_name,
			last_name,
			phone_number,
			email,
			gender,
			password,
			username,
			profile_image_url,
			type,
			created_at
		FROM users
		WHERE id=$1
	`

	row := ur.db.QueryRow(query, id)
	err := row.Scan(
		&result.ID,
		&result.FirstName,
		&result.LastName,
		&phoneNumber,
		&result.Email,
		&gender,
		&result.Password,
		&username,
		&profileImageUrl,
		&result.Type,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	result.PhoneNumber = phoneNumber.String
	result.Gender = gender.String
	result.Username = username.String
	result.ProfileImageUrl = profileImageUrl.String

	return &result, nil
}

func (ur *userRepo) GetAll(params *repo.GetAllUsersParams) (*repo.GetAllUsersResult, error) {
	result := repo.GetAllUsersResult{
		Users: make([]*repo.User, 0),
	}

	offset := (params.Page - 1) * params.Limit

	limit := fmt.Sprintf(" LIMIT %d OFFSET %d ", params.Limit, offset)

	filter := ""
	if params.Search != "" {
		str := "%" + params.Search + "%"
		filter += fmt.Sprintf(`
			WHERE first_name ILIKE '%s' OR last_name ILIKE '%s' OR email ILIKE '%s' 
				OR username ILIKE '%s' OR phone_number ILIKE '%s'`,
			str, str, str, str, str,
		)
	}

	query := `
		SELECT
			id,
			first_name,
			last_name,
			phone_number,
			email,
			gender,
			password,
			username,
			profile_image_url,
			type,
			created_at
		FROM users
		` + filter + `
		ORDER BY created_at desc
		` + limit

	rows, err := ur.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			u                                              repo.User
			phoneNumber, gender, username, profileImageUrl sql.NullString
		)

		err := rows.Scan(
			&u.ID,
			&u.FirstName,
			&u.LastName,
			&phoneNumber,
			&u.Email,
			&gender,
			&u.Password,
			&username,
			&profileImageUrl,
			&u.Type,
			&u.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		u.PhoneNumber = phoneNumber.String
		u.Gender = gender.String
		u.Username = username.String
		u.ProfileImageUrl = profileImageUrl.String

		result.Users = append(result.Users, &u)
	}

	queryCount := `SELECT count(1) FROM users ` + filter
	err = ur.db.QueryRow(queryCount).Scan(&result.Count)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (ur *userRepo) GetByEmail(email string) (*repo.User, error) {
	var (
		result                                         repo.User
		phoneNumber, gender, username, profileImageUrl sql.NullString
	)

	query := `
		SELECT
			id,
			first_name,
			last_name,
			phone_number,
			email,
			gender,
			password,
			username,
			profile_image_url,
			type,
			created_at
		FROM users
		WHERE email=$1
	`

	row := ur.db.QueryRow(query, email)
	err := row.Scan(
		&result.ID,
		&result.FirstName,
		&result.LastName,
		&phoneNumber,
		&result.Email,
		&gender,
		&result.Password,
		&username,
		&profileImageUrl,
		&result.Type,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	result.PhoneNumber = phoneNumber.String
	result.Gender = gender.String
	result.Username = username.String
	result.ProfileImageUrl = profileImageUrl.String

	return &result, nil
}

func (ur *userRepo) UpdatePassword(req *repo.UpdatePassword) error {
	query := `UPDATE users SET password=$1 WHERE id=$2`

	_, err := ur.db.Exec(query, req.Password, req.UserID)
	if err != nil {
		return err
	}

	return nil
}

func (ur *userRepo) Delete(id int64) error {
	query := `DELETE FROM users WHERE id=$1`

	result, err := ur.db.Exec(query, id)
	if err != nil {
		return err
	}

	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (ur *userRepo) Update(user *repo.User) (*repo.User, error) {
	query := `
		UPDATE users SET
			first_name=$1,
			last_name=$2,
			phone_number=$3,
			gender=$4,
			username=$5,
			profile_image_url=$6
		WHERE id=$7
		RETURNING 
			email,
			type,
			created_at
	`

	err := ur.db.QueryRow(
		query,
		user.FirstName,
		user.LastName,
		user.PhoneNumber,
		user.Gender,
		user.Username,
		user.ProfileImageUrl,
		user.ID,
	).Scan(
		&user.Email,
		&user.Type,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}
