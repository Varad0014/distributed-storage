package model

type File struct{
	Id string `json:"id"`
	Name string `json:"name"`
	Size uint64 `json:"size"`
}