package blevebench

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/blevesearch/bleve"
)

type BenchConfig struct {
	IndexType string                 `json:"index_type"`
	KVStore   string                 `json:"kvstore"`
	KVConfig  map[string]interface{} `json:"kvconfig"`
}

func LoadConfigFile(path string) *BenchConfig {
	benchConfig := BenchConfig{
		IndexType: bleve.Config.DefaultIndexType,
		KVStore:   bleve.Config.DefaultKVStore,
		KVConfig:  map[string]interface{}{},
	}
	if path != "" {
		configBytes, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(configBytes, &benchConfig)
		if err != nil {
			log.Fatal(err)
		}
	}
	return &benchConfig
}
