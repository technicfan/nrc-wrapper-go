package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Resource interface {
	Url() string
	Path() string
	Filename() string
	ExpectedHash() string
	HashObj() hash.Hash
	Download() error
}

type NrcResource interface {
	Resource
	Type() int
	IndexPair() Pair
}

func Download(resource Resource) error {
	os.MkdirAll(filepath.Dir(resource.Path()), os.ModePerm)

	response, err := http.Get(resource.Url())
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %v", response.StatusCode)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	expected_hash := resource.ExpectedHash()
	if expected_hash != "" {
		hash := resource.HashObj()

		if _, err := hash.Write(body); err != nil {
			return err
		}
		if hex.EncodeToString(hash.Sum(nil)) != expected_hash {
			return errors.New("wrong hash")
		}
	}

	file, err := os.Create(resource.Path())
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return err
	}

	return nil
}

func DownloadAsync(
	resource NrcResource,
	eofd bool,
	notify bool,
	mods chan <- Pair,
	assets chan <- Pair,
	wg *sync.WaitGroup,
	limiter chan struct{},
) {
	defer wg.Done()
	limiter <- struct{}{}
	defer func() { <-limiter }()

	err := resource.Download()
	if err != nil {
		Notify(
			fmt.Sprintf("Failed to download %s: %s", resource.Filename(), err.Error()),
			eofd,
			notify,
		)
	} else {
		switch resource.Type() {
		case 0:
			mods <- resource.IndexPair()
		case 1:
			assets <- resource.IndexPair()
		}
	}
}
