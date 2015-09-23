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
var count = flag.Int("count", 100000, "total number of documents to process")
var batchSize = flag.Int("batch", 100, "batch size")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile at end")
var numIndexers = flag.Int("numIndexers", 8, "number of indexing goroutines")
var numAnalyzers = flag.Int("numAnalyzers", 8, "number of analyzer goroutines")
var printTime = flag.Duration("printTime", 0*time.Second, "print stats every printTime")
var printCount = flag.Int("printCount", 1000, "print stats every printCount docs")
var bindHttp = flag.String("bindHttp", ":1234", "http bind port")

var totalIndexed uint64
var totalPlainTextIndexed uint64
var start time.Time

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

	bleve.Config.SetAnalysisQueueSize(*numAnalyzers)

	mapping := blevebench.BuildArticleMapping()
	benchConfig := blevebench.LoadConfigFile(*config)

	fmt.Printf("Using Index Type: %s\n", benchConfig.IndexType)
	fmt.Printf("Using KV store: %s\n", benchConfig.KVStore)
	fmt.Printf("Using KV config: %#v\n", benchConfig.KVConfig)
	index, err := bleve.NewUsing(*target, mapping, benchConfig.IndexType, benchConfig.KVStore, benchConfig.KVConfig)
	if err != nil {
		log.Fatal(err)
	}

	start = time.Now()
	work := make(chan *Work)

	//start workers
	var wg sync.WaitGroup
	for i := 0; i < *numIndexers; i++ {
		wg.Add(1)
		go func() {
			batchIndexingWorker(index, work, start)
			wg.Done()
		}()
	}

	// start reading worker
	go readingWorker(index, work)

	// start print time worker
	if *printTime > 0 {
		go printTimeWorker()
	}

	wg.Wait()

	s := index.Stats()
	statsBytes, err := json.Marshal(s)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stats: %s", string(statsBytes))
}

type Work struct {
	batch          *bleve.Batch
	doc            *blevebench.Article
	id             string
	plainTextBytes uint64
}

func printTimeWorker() {
	tickChan := time.NewTicker(*printTime).C
	lastPrintTime := start
	lastBytes := uint64(0)
	for {
		select {
		case <-tickChan:
			bytesNow := atomic.LoadUint64(&totalPlainTextIndexed)
			bytesSince := bytesNow - lastBytes
			timeNow := time.Now()
			timeSince := timeNow.Sub(lastPrintTime)
			mb := float64(bytesSince) / 1000000.0
			log.Printf("mb: %f", mb)
			seconds := float64(timeSince) / float64(time.Second)
			log.Printf("s: %f", seconds)
			log.Printf("%d bytes in %d seconds = %fMB/s", bytesSince, timeSince/time.Second, mb/seconds)
			// reset
			lastPrintTime = timeNow
			lastBytes = bytesNow
		}
	}
}

// func readingWorker(index bleve.Index, work chan *blevebench.Article) {
// 	wikiReader, err := blevebench.NewWikiReader(*source)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer wikiReader.Close()

// 	i := 0
// 	a, err := wikiReader.Next()
// 	for a != nil && err == nil && i <= *count {

// 		if i == (*count / 2) {
// 			// dump mem stats if requested
// 			if *memprofile != "" {
// 				f, err := os.Create(*memprofile)
// 				if err != nil {
// 					log.Fatal(err)
// 				}
// 				pprof.WriteHeapProfile(f)
// 			}
// 		}

// 		work <- a
// 		i++

// 		a, err = wikiReader.Next()
// 	}
// 	if err != nil {
// 		log.Fatalf("reading worker fatal: %v", err)
// 	}
// 	close(work)
// }

// func batchIndexingWorker(index bleve.Index, workChan chan *blevebench.Article, start time.Time) {
// 	for {
// 		select {
// 		case work, ok := <-workChan:
// 			if !ok {
// 				return
// 			}
// 			err := index.Index(work.Title, work)
// 			if err != nil {
// 				log.Fatalf("indexer worker fatal: %v", err)
// 			}
// 			elapsedTime := time.Since(start) / time.Millisecond
// 			updatedTotal := atomic.AddUint64(&totalIndexed, uint64(1))
// 			if updatedTotal%100 == 0 {
// 				log.Printf("%d,%d", updatedTotal, elapsedTime)
// 			}
// 		}
// 	}
// }

func readingWorker(index bleve.Index, work chan *Work) {
	wikiReader, err := blevebench.NewWikiReader(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer wikiReader.Close()

	i := 0

	if *batchSize > 1 {
		batch := index.NewBatch()
		bytesInBatch := uint64(0)
		a, err := wikiReader.Next()
		for a != nil && err == nil && i < *count {
			err = batch.Index(strconv.Itoa(i), a)
			i++
			if err != nil {
				break
			}
			bytesInBatch += uint64(len(a.Title))
			bytesInBatch += uint64(len(a.Text))
			if batch.Size() >= *batchSize {
				work <- &Work{
					batch:          batch,
					plainTextBytes: bytesInBatch,
				}
				batch = index.NewBatch()
				bytesInBatch = 0
			}

			a, err = wikiReader.Next()
		}
		if err != nil {
			log.Fatalf("reading worker fatal: %v", err)
		}
		// close last batch
		if batch.Size() > 0 {
			work <- &Work{
				batch:          batch,
				plainTextBytes: bytesInBatch,
			}
		}

	} else {
		a, err := wikiReader.Next()
		for a != nil && err == nil && i <= *count {
			i++
			work <- &Work{
				doc:            a,
				id:             strconv.Itoa(i),
				plainTextBytes: uint64(len(a.Title) + len(a.Text)),
			}
			a, err = wikiReader.Next()
		}
		if err != nil {
			log.Fatalf("reading worker fatal: %v", err)
		}
	}

	close(work)

	// dump mem stats if requested
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
	}
}

func batchIndexingWorker(index bleve.Index, workChan chan *Work, start time.Time) {
	for {
		select {
		case work, ok := <-workChan:
			if !ok {
				return
			}
			workSize := 1
			if work.batch != nil {
				err := index.Batch(work.batch)
				if err != nil {
					log.Fatalf("indexer worker fatal: %v", err)
				}
				workSize = work.batch.Size()
			} else {
				err := index.Index(work.id, work.doc)
				if err != nil {
					log.Fatalf("indexer worker fatal: %v", err)
				}
			}
			elapsedTime := time.Since(start) / time.Millisecond
			updatedTotal := atomic.AddUint64(&totalIndexed, uint64(workSize))
			atomic.AddUint64(&totalPlainTextIndexed, work.plainTextBytes)
			if updatedTotal%uint64(*printCount) == 0 {
				log.Printf("%d,%d", updatedTotal, elapsedTime)
			}
		}
	}
}
