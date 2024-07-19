package iana

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cachedClient struct {
	cacheDirectory string
	remoteClient   Client
	cacheDuration  time.Duration
}

func NewCachedClient(cacheDirectory string, cacheDuration time.Duration) Client {
	return &cachedClient{cacheDirectory: cacheDirectory,
		remoteClient:  NewRemoteClient(),
		cacheDuration: cacheDuration}
}

func (cc *cachedClient) GetAllTlds() ([]string, error) {
	path := cc.cachePath()

	stat, err := os.Stat(path)
	if err != nil {
		return cc.getAndCacheAllTlds(path)
	}

	if stat.ModTime().Add(cc.cacheDuration).After(time.Now()) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		allTlds := make([]string, 0)
		err = json.Unmarshal(data, &allTlds)
		if err != nil {
			return nil, err
		}

		return allTlds, nil
	} else {
		return cc.getAndCacheAllTlds(path)
	}
}

func (cc *cachedClient) cachePath() string {
	return filepath.Join(cc.cacheDirectory, "iana-tlds.json")
}

func (cc *cachedClient) getAndCacheAllTlds(path string) ([]string, error) {
	allTlds, err := cc.remoteClient.GetAllTlds()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(allTlds)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return allTlds, nil
}
