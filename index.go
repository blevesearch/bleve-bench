package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
)

var config = flag.String("config", "", "configuration file to use")
var source = flag.String("source", "tmp/enwiki.txt", "wikipedia line file")
var target = flag.String("target", "bench.bleve", "target index filename")
var count = flag.Int("count", 100000, "total number of documents to process")
var batchSize = flag.Int("batch", 100, "batch size")
var level = flag.Int("level", 1000, "report level")
var qrepeat = flag.Int("qrepeat", 5, "query repeat")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile every level")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	start := time.Now()

	wikiReader, err := NewWikiReader(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer wikiReader.Close()

	mapping := buildArticleMapping()
	storeType := bleve.Config.DefaultKVStore
	storeConfig := map[string]interface{}{}

	if *config != "" {
		configBytes, err := ioutil.ReadFile(*config)
		if err != nil {
			log.Fatal(err)
		}
		var configJSON map[string]interface{}
		err = json.Unmarshal(configBytes, &configJSON)
		if err != nil {
			log.Fatal(err)
		}
		var possibleConfig interface{}
		for storeType, possibleConfig = range configJSON {
			var ok bool
			storeConfig, ok = possibleConfig.(map[string]interface{})
			if !ok {
				log.Fatal("kv config must be map")
			}
			break
		}
	}

	fmt.Printf("Using KV store: %s\n", storeType)
	fmt.Printf("Using KV config: %#v\n", storeConfig)
	index, err := bleve.NewUsing(*target, mapping, storeType, storeConfig)
	if err != nil {
		log.Fatal(err)
	}
	_, store, err := index.Advanced()
	if err != nil {
		log.Fatal(err)
	}

	// print header
	fmt.Printf("elapsed,docs,avg_single_doc_ms,avg_batched_doc_ms,query_water_matches,first_query_water_ms,avg_repeated%d_query_water_ms", *qrepeat)
	printOtherHeader(store)
	fmt.Printf("\n")

	singleCount := 0
	var singleTime time.Duration
	batchCount := 0
	var batchTime time.Duration
	batch := bleve.NewBatch()
	for i := 1; i < (*count)+1; i++ {

		leveli := i % *level

		a, err := wikiReader.Next()
		if err != nil {
			log.Fatal(err)
		}
		if leveli < *batchSize {
			// index single
			singleStart := time.Now()
			err = index.Index(a.Title, a)
			if err != nil {
				log.Fatalf("error indexing: %v", err)
			}
			duration := time.Since(singleStart)
			singleCount++
			singleTime += duration
		} else {
			// add to batch
			batch.Index(a.Title, a)
			// if batch is full index it
			if batch.Size() == *batchSize {
				batchStart := time.Now()
				err := index.Batch(batch)
				if err != nil {
					log.Fatalf("error executing batch: %v", err)
				}
				duration := time.Since(batchStart)
				batchCount++
				batchTime += duration
				// reset batch
				batch = bleve.NewBatch()
			}
		}

		if leveli == 0 {

			// run some queries
			termQueryCount := 0
			termQueryStart := time.Now()
			termQuery := bleve.NewTermQuery("water")
			termQuery.SetField("text")
			termSearch := bleve.NewSearchRequest(termQuery)
			searchResults, err := index.Search(termSearch)
			if err != nil {
				log.Fatalf("error searching: %v", err)
			}
			termQueryCount++
			termQueryTime := time.Since(termQueryStart)

			firstQueryTime := float64(termQueryTime)

			for termQueryCount < *qrepeat {
				termQueryStart = time.Now()
				searchResults, err = index.Search(termSearch)
				if err != nil {
					log.Fatal(err)
				}
				termQueryCount++
				termQueryTime += time.Since(termQueryStart)
			}

			// print stats
			avgSingleDocTime := float64(singleTime) / float64(singleCount)
			avgBatchTime := float64(batchTime) / float64(batchCount)
			avgBatchDocTime := float64(avgBatchTime) / float64(*batchSize)
			avgQueryTime := float64(termQueryTime) / float64(termQueryCount)
			elapsedTime := time.Since(start) / time.Millisecond
			fmt.Printf("%d,%d,%f,%f,%d,%f,%f", elapsedTime, i, avgSingleDocTime/float64(time.Millisecond), avgBatchDocTime/float64(time.Millisecond), searchResults.Total, firstQueryTime/float64(time.Millisecond), avgQueryTime/float64(time.Millisecond))
			printOther(store)
			fmt.Printf("\n")

			// reset stats
			singleCount = 0
			singleTime = 0
			batchCount = 0
			batchTime = 0

			// dump mem stats if requested
			if *memprofile != "" {
				f, err := os.Create(strconv.Itoa(i) + "-" + *memprofile)
				if err != nil {
					log.Fatal(err)
				}
				pprof.WriteHeapProfile(f)
			}
		}

	}
}
