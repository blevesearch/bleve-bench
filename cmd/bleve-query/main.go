package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve-bench"

	_ "github.com/blevesearch/bleve/config"
)

var config = flag.String("config", "", "configuration file to use")
var source = flag.String("source", "../../tmp/enwiki.txt", "wikipedia line file")
var target = flag.String("target", "bench.bleve", "target index filename")
var count = flag.Int("count", 1000, "total number of documents to process")
var qcount = flag.Int("querycount", 100000, "total number of query to process")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile at end")
var numQueryThreads = flag.Int("queryThreads", 8, "number of querying goroutines")
var printTime = flag.Duration("printTime", 0*time.Second, "print stats every printTime")
var bindHttp = flag.String("bindHttp", ":1234", "http bind port")

var totalRequests uint64
var totalTimeTaken uint64

type queryFunc func() *bleve.SearchRequest

var queryType map[string]queryFunc = map[string]queryFunc{
	"term":  buildTermQuery,
	"match": buildMatchQuery,
	"fuzzy": buildFuzzyQuery,
}

func main() {
	flag.Parse()

	go http.ListenAndServe(*bindHttp, nil)

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	bleve.Config.SetAnalysisQueueSize(8)

	mapping := blevebench.BuildArticleMapping()
	benchConfig := blevebench.LoadConfigFile(*config)

	fmt.Printf("Using Index Type: %s\n", benchConfig.IndexType)
	fmt.Printf("Using KV store: %s\n", benchConfig.KVStore)
	fmt.Printf("Using KV config: %#v\n", benchConfig.KVConfig)
	index, err := bleve.NewUsing(*target, mapping, benchConfig.IndexType, benchConfig.KVStore, benchConfig.KVConfig)
	if err != nil {
		log.Fatal(err)
	}

	// start reading worker
	indexWorker(index)

	resetChan := make(chan bool)
	if *printTime > 0 {
		go printTimeWorker(resetChan)
	}

	for s, h := range queryType {
		log.Println("running", s, "query")
		var wg sync.WaitGroup
		q := h()
		start := time.Now()
		// Start the query after indexing
		for i := 0; i < *numQueryThreads; i++ {
			wg.Add(1)
			go func() {
				docs := *qcount / (*numQueryThreads)
				if i == *numQueryThreads-1 {
					docs = docs + *qcount%(*numQueryThreads)
				}
				queryWorker(index, q, docs)
				wg.Done()
			}()
		}
		wg.Wait()
		end := time.Now()
		timeTaken := end.Sub(start)
		seconds := float64(timeTaken) / float64(time.Second)
		mb := int(float64(*qcount) / float64(seconds))
		log.Println("Result:", s, "query - queries per second", mb)
		resetChan <- true
	}
	s := index.Stats()
	statsBytes, err := json.Marshal(s)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stats: %s", string(statsBytes))
}

func printTimeWorker(r chan bool) {
	tickChan := time.NewTicker(*printTime).C
	var lastR uint64
	lastT := time.Now()
	for {
		select {
		case <-tickChan:
			queries := atomic.LoadUint64(&totalRequests) - lastR
			//timeTaken := atomic.LoadUint64(&totalTimeTaken) - lastT
			timeNow := time.Now()
			timeTaken := timeNow.Sub(lastT)
			seconds := float64(timeTaken) / float64(time.Second)
			mb := int(float64(queries) / float64(seconds))
			log.Println("Total queries", totalRequests, "queries per second", mb)
			// reset
			lastR = totalRequests
			lastT = timeNow
		case <-r:
			totalRequests = 0
			lastR = 0
			lastT = time.Now()
		}
	}
}

func indexWorker(index bleve.Index) {
	wikiReader, err := blevebench.NewWikiReader(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer wikiReader.Close()
	i := 0
	a, err := wikiReader.Next()
	for a != nil && err == nil && i <= *count {
		i++
		index.Index(strconv.Itoa(i), a)
		a, err = wikiReader.Next()
	}
	if err != nil {
		log.Fatalf("reading worker fatal: %v", err)
	}

	// dump mem stats if requested
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
	}
}

func queryWorker(index bleve.Index, sr *bleve.SearchRequest, repeat int) {
	termQueryCount := 0
	for termQueryCount < repeat {
		termQueryStart := time.Now()
		_, err := index.Search(sr)
		if err != nil {
			log.Fatal(err)
		}
		atomic.AddUint64(&totalRequests, 1)
		atomic.AddUint64(&totalTimeTaken, uint64(time.Since(termQueryStart)))
		termQueryCount++
	}
}

func buildTermQuery() *bleve.SearchRequest {
	termQuery := bleve.NewTermQuery("water")
	termQuery.SetField("text")
	termSearch := bleve.NewSearchRequest(termQuery)
	return termSearch
}

func buildMatchQuery() *bleve.SearchRequest {
	termQuery := bleve.NewTermQuery("water")
	termQuery.SetField("text")
	termSearch := bleve.NewSearchRequest(termQuery)
	return termSearch
}

func buildFuzzyQuery() *bleve.SearchRequest {
	termQuery := bleve.NewFuzzyQuery("wate")
	termQuery.SetField("text")
	termSearch := bleve.NewSearchRequest(termQuery)
	return termSearch
}
