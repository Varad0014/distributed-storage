package errors

import "errors"

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrInvalidFileName   = errors.New("invalid file name")
	ErrFileTooLarge      = errors.New("file too large")
)
