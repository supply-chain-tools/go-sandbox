package iana

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client interface {
	GetAllTlds() ([]string, error)
}

type remoteClient struct {
}

func NewRemoteClient() Client {
	return &remoteClient{}
}

func (rc *remoteClient) GetAllTlds() ([]string, error) {
	data, err := getUrl("https://data.iana.org/TLD/tlds-alpha-by-domain.txt")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	result := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		result = append(result, strings.ToLower(line))
	}

	return result, nil
}

func getUrl(url string) ([]byte, error) {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code not 200 while fetching url '%s'", url)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
