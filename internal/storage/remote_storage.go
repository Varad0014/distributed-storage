package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type RemoteStorage struct {
	baseURL string
	token   string
	client  *http.Client
}

type putResponse struct {
	ID   string `json:"id"`
	Size uint64 `json:"size"`
}

type checksumResponse struct {
	Checksum string `json:"checksum"`
}

func NewRemoteStorage(baseURL string, token string) *RemoteStorage {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
	}

	return &RemoteStorage{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Minute,
		},
	}
}

func (rs *RemoteStorage) Save(id string, srcFile io.Reader) (uint64, error) {
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)

	req, err := rs.newRequest(http.MethodPost, url, srcFile)
	if err != nil {
		return 0, fmt.Errorf("create save request for object %s: %w", id, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("save object %s to %s: %w", id, rs.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf(
			"save object %s to %s: storage node returned status %d",
			id,
			rs.baseURL,
			resp.StatusCode,
		)
	}

	var response putResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf(
			"decode save response for object %s from %s: %w",
			id,
			rs.baseURL,
			err,
		)
	}

	return response.Size, nil
}

func (rs *RemoteStorage) Open(id string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)

	req, err := rs.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create open request for object %s: %w", id, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open object %s from %s: %w", id, rs.baseURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf(
				"open object %s from %s: %w",
				id,
				rs.baseURL,
				os.ErrNotExist,
			)
		}

		return nil, fmt.Errorf(
			"open object %s from %s: storage node returned status %d",
			id,
			rs.baseURL,
			resp.StatusCode,
		)
	}

	// The caller must close the response body.
	return resp.Body, nil
}

func (rs *RemoteStorage) Delete(id string) error {
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)

	req, err := rs.newRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create delete request for object %s: %w", id, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete object %s from %s: %w", id, rs.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf(
			"delete object %s from %s: %w",
			id,
			rs.baseURL,
			os.ErrNotExist,
		)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"delete object %s from %s: storage node returned status %d",
			id,
			rs.baseURL,
			resp.StatusCode,
		)
	}

	return nil
}

func (rs *RemoteStorage) Exists(id string) (bool, error) {
	url := fmt.Sprintf("%s/objects/%s", rs.baseURL, id)

	req, err := rs.newRequest(http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Errorf("create existence request for object %s: %w", id, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return false, fmt.Errorf(
			"check existence of object %s on %s: %w",
			id,
			rs.baseURL,
			err,
		)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil

	case http.StatusNotFound:
		return false, nil

	default:
		return false, fmt.Errorf(
			"check existence of object %s on %s: storage node returned status %d",
			id,
			rs.baseURL,
			resp.StatusCode,
		)
	}
}

func (rs *RemoteStorage) Checksum(id string) (string, error) {
	url := fmt.Sprintf("%s/objects-checksum/%s", rs.baseURL, id)

	req, err := rs.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create checksum request for object %s: %w", id, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"get checksum for object %s from %s: %w",
			id,
			rs.baseURL,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf(
			"get checksum for object %s from %s: %w",
			id,
			rs.baseURL,
			os.ErrNotExist,
		)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf(
			"get checksum for object %s from %s: storage node returned status %d",
			id,
			rs.baseURL,
			resp.StatusCode,
		)
	}

	var response checksumResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf(
			"decode checksum response for object %s from %s: %w",
			id,
			rs.baseURL,
			err,
		)
	}

	return response.Checksum, nil
}

func (rs *RemoteStorage) HealthCheck() error {
	url := rs.baseURL + "/health"

	req, err := rs.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create health-check request for %s: %w", rs.baseURL, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check for %s: %w", rs.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"health check for %s: storage node returned status %d",
			rs.baseURL,
			resp.StatusCode,
		)
	}

	return nil
}

func (rs *RemoteStorage) List() ([]string, error) {
	url := rs.baseURL + "/objects"

	req, err := rs.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create list request for %s: %w", rs.baseURL, err)
	}

	resp, err := rs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list objects from %s: %w", rs.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"list objects from %s: storage node returned status %d",
			rs.baseURL,
			resp.StatusCode,
		)
	}

	objects := make([]string, 0)

	if err := json.NewDecoder(resp.Body).Decode(&objects); err != nil {
		return nil, fmt.Errorf(
			"decode object list from %s: %w",
			rs.baseURL,
			err,
		)
	}

	return objects, nil
}

func (rs *RemoteStorage) newRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create %s request for %s: %w", method, url, err)
	}

	req.Header.Set("Authorization", "Bearer "+rs.token)

	return req, nil
}

var _ StorageNode = (*RemoteStorage)(nil)
var _ HealthChecker = (*RemoteStorage)(nil)
