package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve-bench"

	_ "github.com/blevesearch/bleve/config"
	_ "github.com/blevesearch/bleve/index/store/null"
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
var printTime = flag.Duration("printTime", 5*time.Second, "print stats every printTime")
var bindHttp = flag.String("bindHttp", ":1234", "http bind port")
var statsFile = flag.String("statsFile", "", "<stdout>")

var totalIndexed uint64
var lastTotalIndexed uint64
var totalPlainTextIndexed uint64
var lastTotalPlainTextIndexed uint64

var timeStart time.Time
var timeLast time.Time

var statsWriter = os.Stdout

func main() {
	flag.Parse()

	go http.ListenAndServe(*bindHttp, nil) // For expvar.

	if *statsFile != "" {
		// create all parents if necessary
		dir := path.Dir(*statsFile)
		os.MkdirAll(dir, 0755)

		var err error
		statsWriter, err = os.Create(*statsFile)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
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

	printHeader()
	timeStart = time.Now()
	timeLast = timeStart
	printLine()

	work := make(chan *Work)

	// start reading worker
	go readingWorker(index, work)

	// start print time worker
	if *printTime > 0 {
		go printTimeWorker()
	}

	// start workers
	var wg sync.WaitGroup
	for i := 0; i < *numIndexers; i++ {
		wg.Add(1)
		go func() {
			batchIndexingWorker(index, work, timeStart)
			wg.Done()
		}()
	}

	wg.Wait()

	// print final stats
	printLine()
}

type Work struct {
	batch          *bleve.Batch
	doc            *blevebench.Article
	id             string
	plainTextBytes uint64
}

func printTimeWorker() {
	tickChan := time.NewTicker(*printTime).C
	for range tickChan {
		printLine()
	}
}

var outputFields = []string{
	"date",
	"docs_indexed",
	"plaintext_bytes_indexed",
	"avg_mb_per_second",
	"mb_per_second",
}

func printHeader() {
	fmt.Fprintf(statsWriter, "%s\n", strings.Join(outputFields, ","))
}

func printLine() {
	// get
	timeNow := time.Now()
	nowTotalIndexed := atomic.LoadUint64(&totalIndexed)
	nowTotalPlainTextIndexed := atomic.LoadUint64(&totalPlainTextIndexed)

	// calculate
	curPlainTextIndexed := nowTotalPlainTextIndexed - lastTotalPlainTextIndexed

	cumTimeTaken := timeNow.Sub(timeStart)
	curTimeTaken := timeNow.Sub(timeLast)

	cumMBytes := float64(nowTotalPlainTextIndexed) / 1000000.0
	curMBytes := float64(curPlainTextIndexed) / 1000000.0

	cumSeconds := float64(cumTimeTaken) / float64(time.Second)
	curSeconds := float64(curTimeTaken) / float64(time.Second)

	dateNow := timeNow.Format(time.RFC3339)
	fmt.Fprintf(statsWriter, "%s,%d,%d,%f,%f\n", dateNow, nowTotalIndexed,
		nowTotalPlainTextIndexed, cumMBytes/cumSeconds, curMBytes/curSeconds)

	timeLast = timeNow
	lastTotalIndexed = nowTotalIndexed
	lastTotalPlainTextIndexed = nowTotalPlainTextIndexed
}

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
		f.Close()
	}
}

func batchIndexingWorker(index bleve.Index, workChan chan *Work, timeStart time.Time) {
	for work := range workChan {
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
		atomic.AddUint64(&totalIndexed, uint64(workSize))
		atomic.AddUint64(&totalPlainTextIndexed, work.plainTextBytes)
	}
}
