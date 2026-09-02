package repository

import (
	"context"
	"errors"
	"fmt"
	appErrors "github.com/Varad0014/distributed-storage/internal/errors"
	"github.com/Varad0014/distributed-storage/internal/model"
	"github.com/jackc/pgx/v5"
)

type FileRepository struct {
	db *pgx.Conn
}

func NewFileRepository(db *pgx.Conn) *FileRepository {
	return &FileRepository{db: db}
}

func (fr *FileRepository) CreateTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS files (
		id UUID PRIMARY KEY,
		name TEXT NOT NULL,
		size BIGINT NOT NULL,
		checksum TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);`
	_, err := fr.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf(
			"create files table: %w",
			err,
		)
	}
	return nil
}

func (fr *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `INSERT INTO files (id, name, size, checksum, created_at) VALUES ($1, $2, $3, $4, $5);`
	_, err := fr.db.Exec(ctx, query, file.Id, file.Name, file.Size, file.Checksum, file.CreatedAt)
	if err != nil {
		return fmt.Errorf(
			"create file metadata %s: %w",
			file.Id,
			err,
		)
	}

	return nil
}

func (fr *FileRepository) GetAll(ctx context.Context) ([]model.File, error) {
	query := `SELECT id, name, size, checksum, created_at FROM files ORDER BY created_at DESC;`
	rows, err := fr.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"query file metadata: %w",
			err,
		)
	}
	defer rows.Close()

	// dont use var files []model.File, because it will be nil, and append will create a new slice, but we want to return the same slice
	files := make([]model.File, 0)
	for rows.Next() {
		var file model.File
		err := rows.Scan(&file.Id, &file.Name, &file.Size, &file.Checksum, &file.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf(
				"scan file metadata: %w",
				err,
			)
		}
		files = append(files, file)
	}

	// abnormal or normal return of rows.Next() loop, check for errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate file metadata: %w",
			err,
		)
	}
	return files, nil
}

func (fr *FileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	query := `SELECT id, name, size, checksum, created_at FROM files WHERE id = $1;`
	var file model.File
	err := fr.db.QueryRow(ctx, query, id).Scan(&file.Id, &file.Name, &file.Size, &file.Checksum, &file.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.ErrFileNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"get file metadata %s: %w",
			id,
			err,
		)
	}

	return &file, nil
}

func (fr *FileRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1;`
	result, err := fr.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf(
			"delete file metadata %s: %w",
			id,
			err,
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.ErrFileNotFound
	}
	return nil
}
