package utils

import (
	"encoding/json"
	"io"
	"os"
)

type Index map[string]map[string]string

type Pair struct {
	Key   string
	Value map[string]string
}

func Read_index(path string) Index {
	data := make(Index)
	file, err := os.Open(path)
	if err != nil {
		return data
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return data
	}
	defer file.Close()

	err = json.Unmarshal(byte_data, &data)
	if err != nil {
		return data
	}

	return data
}

func (data Index) Write(path string) error {
	var file *os.File
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	json_string, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func (data Index) Merge(index chan Pair) Index {
	for e := range index {
		data[e.Key] = e.Value
	}
	return data
}
