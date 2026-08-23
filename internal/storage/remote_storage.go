package storage

import (
	"fmt"
	"os"
	"net/http"
	"io"
	"encoding/json"
)

type RemoteStorage struct{
	baseURL string
	client *http.Client
}

type putResponse struct{
	Id string `json:"id"`
	Size uint64 `json:"size"`
}

type checksumResponse struct{
	Checksum string `json:"checksum"`
}

func NewRemoteStorage(baseURL string) *RemoteStorage{
	return &RemoteStorage{
		baseURL: baseURL,
		client: &http.Client{},
	}
}



func (rs *RemoteStorage) Save(id string, srcFile io.Reader)(uint64, error){
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)
	req, err := http.NewRequest(http.MethodPost, url, srcFile)
	if err != nil{
		return 0, err
	}
	resp, err := rs.client.Do(req)
	if err != nil{
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300{
		return 0, fmt.Errorf("failed to save object, status code: %d", resp.StatusCode)
	}
	var response putResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil{
		return 0, err
	}
	return response.Size, nil
}

func (rs *RemoteStorage) Open(id string)(io.ReadCloser, error){
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)
	resp, err := rs.client.Get(url)
	if err != nil{
		return nil, err
	}
	if resp.StatusCode != http.StatusOK{
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound{
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to open object, status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (rs *RemoteStorage) Delete(id string) error{
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil{
		// fmt.Println(err)
		return err
	}
	resp, err := rs.client.Do(req)
	if err != nil{
		// fmt.Println(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound{
		fmt.Println(resp.StatusCode)
		return os.ErrNotExist
	}
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300{
		return fmt.Errorf("failed to delete object, status code: %d", resp.StatusCode)
	}
	return nil
}

func (rs *RemoteStorage) Exists(id string) (bool, error){
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil{
		return false, err
	}
	resp, err := rs.client.Do(req)
	if err != nil{
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode{
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("failed to check object existence, status code: %d", resp.StatusCode)
	}
}

func (rs *RemoteStorage) Checksum(id string) (string, error){
	url := fmt.Sprintf("%s/objects-checksum/%s", rs.baseURL, id)
	resp, err := rs.client.Get(url)
	if err != nil{
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound{
		return "", os.ErrNotExist
	}
		
	if resp.StatusCode < 200 || resp.StatusCode >= 300{
		return "", fmt.Errorf("failed to get checksum, status code: %d", resp.StatusCode)
	}
	var checksumResp checksumResponse
	if err := json.NewDecoder(resp.Body).Decode(&checksumResp); err != nil{
		return "", err
	}
	return checksumResp.Checksum, nil
}

var _ StorageNode = (*RemoteStorage)(nil)

