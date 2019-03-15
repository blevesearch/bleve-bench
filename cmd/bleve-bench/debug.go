//  Copyright (c) 2019 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
