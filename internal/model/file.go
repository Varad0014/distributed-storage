package model

import ("time")

type File struct{
	Id string `json:"id"`
	Name string `json:"name"`
	Size uint64 `json:"size"`
	Checksum string `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
}