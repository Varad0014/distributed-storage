package repository

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/Varad0014/distributed-storage/internal/model"
)

type FileRepository struct{
	db *pgx.Conn
}

func NewFileRepository(db *pgx.Conn)(*FileRepository){
	return &FileRepository{db: db}
}

func (fr *FileRepository) CreateTable(ctx context.Context) error{
	query := `CREATE TABLE IF NOT EXISTS files (
		id UUID PRIMARY KEY,
		name TEXT NOT NULL,
		size BIGINT NOT NULL,
		checksum TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);`
	_, err := fr.db.Exec(ctx, query)
	return err
}

func (fr *FileRepository) Create(ctx context.Context, file *model.File) error{
	query := `INSERT INTO files (id, name, size, checksum, created_at) VALUES ($1, $2, $3, $4, $5);`
	_, err := fr.db.Exec(ctx, query, file.Id, file.Name, file.Size, file.Checksum, file.CreatedAt)
	return err
}		

func (fr *FileRepository) GetAll(ctx context.Context) ([]model.File, error){
	query := `SELECT id, name, size, checksum, created_at FROM files ORDER BY created_at DESC;`
	rows, err := fr.db.Query(ctx, query)
	if err != nil{
		return nil, err
	}
	defer rows.Close()

	// dont use var files []model.File, because it will be nil, and append will create a new slice, but we want to return the same slice
	files := make([]model.File, 0)
	for rows.Next(){
		var file model.File
		err := rows.Scan(&file.Id, &file.Name, &file.Size, &file.Checksum, &file.CreatedAt)
		if err != nil{
			return nil, err
		}
		files = append(files, file)
	}

	// abnormal or normal return of rows.Next() loop, check for errors
	if err := rows.Err(); err != nil{
		return nil, err
	}
	return files, nil
}

func (fr *FileRepository) GetByID(ctx context.Context, id string) (*model.File, error){
	query := `SELECT id, name, size, checksum, created_at FROM files WHERE id = $1;`
	var file model.File
	err := fr.db.QueryRow(ctx, query, id).Scan(&file.Id, &file.Name, &file.Size, &file.Checksum, &file.CreatedAt)
	if err != nil{
		return nil, err
	}
	return &file, nil
}

func (fr *FileRepository) Delete(ctx context.Context, id string) error{
	query := `DELETE FROM files WHERE id = $1;`
	_, err := fr.db.Exec(ctx, query, id)
	return err
}