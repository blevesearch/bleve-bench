package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/config"
	"github.com/blevesearch/bleve/search/query"
)

var target = flag.String("index", "bench.bleve", "index filename")
var bindHTTP = flag.String("bindHttp", ":1234", "http bind port")
var statsFile = flag.String("statsFile", "", "<stdout>")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write mem profile to file")
var qtype = flag.String("queryType", "term", "type of query to execute: term, prefix, query_string")
var qfield = flag.String("field", "text", "the field to query, not applicable to query_string queries")
var qclients = flag.Int("clients", 1, "the number of query clients")
var qtime = flag.Duration("time", 1*time.Minute, "time to run the test")
var printTime = flag.Duration("printTime", 5*time.Second, "print stats every printTime")
var traceprofile = flag.String("traceprofile", "", "write trace profile to file")

var statsWriter = os.Stdout

var queriesStarted uint64
var queriesFinished uint64
var lastQueriesFinished uint64
var timeStart time.Time
var timeLast time.Time

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

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.WriteHeapProfile(f)
	}

	if *traceprofile != "" {
		f, err := os.Create(*traceprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		trace.Start(f)
		defer trace.Stop()
	}

	if flag.NArg() < 1 {
		log.Fatal("must suppy at least 1 query")
	}

	queries := make([]query.Query, flag.NArg())
	for i, arg := range flag.Args() {
		var query query.Query

		switch *qtype {
		case "prefix":
			pquery := bleve.NewPrefixQuery(arg)
			if *qfield != "" {
				pquery.SetField(*qfield)
			}
			query = pquery
		case "term":
			pquery := bleve.NewTermQuery(arg)
			if *qfield != "" {
				pquery.SetField(*qfield)
			}
			query = pquery
		case "query_string":
			// build a search with the provided parameters
			query = bleve.NewQueryStringQuery(arg)
		}

		queries[i] = query
	}

	index, err := bleve.Open(*target)
	if err != nil {
		log.Fatal(err)
	}

	closeChan := make(chan struct{})
	time.AfterFunc(*qtime, func() {
		close(closeChan)
	})

	var wg sync.WaitGroup
	for i := 0; i < *qclients; i++ {
		wg.Add(1)
		go func() {
			queryClient(index, queries, closeChan)
			wg.Done()
		}()
	}

	printHeader()
	timeStart = time.Now()
	timeLast = timeStart
	printLine()

	// start print time worker
	if *printTime > 0 {
		go printTimeWorker()
	}

	wg.Wait()

	// print final stats
	printLine()

	index.Close()
}

func queryClient(index bleve.Index, queries []query.Query, closeChan chan struct{}) {

	// query client first creates its own unique order to run the queries
	perm := rand.Perm(len(queries))
	i := 0
	for {
		select {
		case <-closeChan:
			return
		default:
			qi := i % len(queries)
			i++
			p := perm[qi]
			q := queries[p]
			atomic.AddUint64(&queriesStarted, 1)
			req := bleve.NewSearchRequest(q)
			_, err := index.Search(req)
			if err != nil {
				log.Fatal(err)
			}
			atomic.AddUint64(&queriesFinished, 1)
		}
	}
}

var outputFields = []string{
	"date",
	"queries_finished",
	"avg_queries_per_second",
	"queries_per_second",
}

func printHeader() {
	fmt.Fprintf(statsWriter, "%s\n", strings.Join(outputFields, ","))
}

func printLine() {
	// get
	timeNow := time.Now()
	nowQueriesFinished := atomic.LoadUint64(&queriesFinished)

	// calculate
	curQueriesFinished := nowQueriesFinished - lastQueriesFinished

	cumTimeTaken := timeNow.Sub(timeStart)
	curTimeTaken := timeNow.Sub(timeLast)

	cumSeconds := float64(cumTimeTaken) / float64(time.Second)
	curSeconds := float64(curTimeTaken) / float64(time.Second)

	dateNow := timeNow.Format(time.RFC3339)
	fmt.Fprintf(statsWriter, "%s,%d,%f,%f\n", dateNow, nowQueriesFinished,
		float64(nowQueriesFinished)/cumSeconds, float64(curQueriesFinished)/curSeconds)

	timeLast = timeNow
	lastQueriesFinished = nowQueriesFinished
}

func printTimeWorker() {
	tickChan := time.NewTicker(*printTime).C
	for range tickChan {
		printLine()
	}
}
