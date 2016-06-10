package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
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

	"github.com/blevesearch/bleve/analysis"
	_ "github.com/blevesearch/bleve/config"
	_ "github.com/blevesearch/bleve/index/store/metrics"
	_ "github.com/blevesearch/bleve/index/store/null"
)

var analyzerName = flag.String("analyzer", "standard", "analyzer to use")
var source = flag.String("source", "../../tmp/enwiki.txt", "wikipedia line file")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile at end")
var readerQueueSize = flag.Int("readerQueueSize", 8, "size of queue output from reader")
var numAnalyzers = flag.Int("numAnalyzers", 8, "number of analyzer goroutines")
var maxTextSize = flag.Int("maxTextSize", 0, "when > 0, text is clipped to this length")
var printTime = flag.Duration("printTime", 5*time.Second, "print stats every printTime")
var bindHTTP = flag.String("bindHttp", ":1234", "http bind port")
var count = flag.Int("count", 100000, "total number of documents to process")
var statsFile = flag.String("statsFile", "", "<stdout>")

var tokensProduced uint64
var lastTokensProduced uint64

var timeStart time.Time
var timeLast time.Time

var statsWriter = os.Stdout

func main() {
	flag.Parse()

	go http.ListenAndServe(*bindHTTP, nil) // For expvar.

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

	defineCustomAnalyzers()

	analyzer, err := bleve.Config.Cache.AnalyzerNamed(*analyzerName)
	if err != nil {
		log.Fatal(err)
	}

	printHeader()
	timeStart = time.Now()
	timeLast = timeStart
	printLine()

	work := make(chan *work, *readerQueueSize)

	// start reading worker
	go readingWorker(work)

	// start print time worker
	if *printTime > 0 {
		go printTimeWorker()
	}

	// start workers
	var wg sync.WaitGroup
	for i := 0; i < *numAnalyzers; i++ {
		wg.Add(1)
		go func() {
			analyzerWorker(analyzer, work, timeStart)
			wg.Done()
		}()
	}

	wg.Wait()

	// print final stats
	printLine()

}

var outputFields = []string{
	"date",
	"tokens",
	"avg_million_tokens_per_second",
	"milllion_tokens_per_second",
}

func printHeader() {
	fmt.Fprintf(statsWriter, "%s\n", strings.Join(outputFields, ","))
}

func printLine() {
	// get
	timeNow := time.Now()
	nowTokensProduced := atomic.LoadUint64(&tokensProduced)

	// calculate
	curTokensProduced := nowTokensProduced - lastTokensProduced

	cumTimeTaken := timeNow.Sub(timeStart)
	curTimeTaken := timeNow.Sub(timeLast)

	cumSeconds := float64(cumTimeTaken) / float64(time.Second)
	curSeconds := float64(curTimeTaken) / float64(time.Second)

	dateNow := timeNow.Format(time.RFC3339)
	fmt.Fprintf(statsWriter, "%s,%d,%f,%f\n", dateNow, nowTokensProduced,
		float64(nowTokensProduced/1000000)/cumSeconds, float64(curTokensProduced/1000000)/curSeconds)

	timeLast = timeNow
	lastTokensProduced = nowTokensProduced
}

func printTimeWorker() {
	tickChan := time.NewTicker(*printTime).C
	for range tickChan {
		printLine()
	}
}

type work struct {
	doc            *blevebench.Article
	id             string
	plainTextBytes uint64
}

func readingWorker(w chan *work) {
	wikiReader, err := blevebench.NewWikiReader(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer wikiReader.Close()

	i := 0

	a, err := wikiReader.Next()
	for a != nil && err == nil && i <= *count {
		if *maxTextSize > 0 && len(a.Text) > *maxTextSize {
			a.Text = a.Text[0:*maxTextSize]
		}

		i++
		w <- &work{
			doc:            a,
			id:             strconv.Itoa(i),
			plainTextBytes: uint64(len(a.Title) + len(a.Text)),
		}
		a, err = wikiReader.Next()
	}
	if err != nil {
		log.Fatalf("reading worker fatal: %v", err)
	}

	close(w)
}

func analyzerWorker(analyzer *analysis.Analyzer, workChan chan *work, timeStart time.Time) {
	for work := range workChan {

		ts := analyzer.Analyze([]byte(work.doc.Text))
		atomic.AddUint64(&tokensProduced, uint64(len(ts)))
	}
}

// replicate some analyzers used in luceneutil
func defineCustomAnalyzers() {

	_, err := bleve.Config.Cache.DefineTokenFilter("edgeNGram13", map[string]interface{}{
		"edge": `front`,
		"min":  1.0,
		"max":  3.0,
		"type": `edge_ngram`,
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = bleve.Config.Cache.DefineAnalyzer("edgeNGrams", map[string]interface{}{
		"type":      "custom",
		"tokenizer": "whitespace",
		"token_filters": []string{
			"edgeNGram13",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = bleve.Config.Cache.DefineTokenFilter("shingle22", map[string]interface{}{
		"min":       2.0,
		"max":       2.0,
		"type":      `shingle`,
		"separator": ``,
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = bleve.Config.Cache.DefineAnalyzer("shingles", map[string]interface{}{
		"type":      "custom",
		"tokenizer": "whitespace",
		"token_filters": []string{
			"shingle22",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
