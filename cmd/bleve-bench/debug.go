// +build debug

package main

import (
	"fmt"
	"os"

	"github.com/blevesearch/bleve/index/store"
	"github.com/blevesearch/bleve/index/store/metrics"
)

func printOtherHeader(s store.KVStore) {
	storeMetrics, ok := s.(*metrics.Store)
	if ok && storeMetrics != nil {
		fmt.Printf(",")
		storeMetrics.WriteCSVHeader(os.Stdout)
	}
}

func printOther(s store.KVStore) {
	storeMetrics, ok := s.(*metrics.Store)
	if ok && storeMetrics != nil {
		fmt.Printf(",")
		storeMetrics.WriteCSV(os.Stdout)
	}
}
