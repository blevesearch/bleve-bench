package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve-bench"
	_ "github.com/blevesearch/bleve/config"
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
var configDir = flag.String("configdir", "", "directory for configs")
var doplot = flag.Bool("plot", false, "generate plots/html")

var typename []string = []string{
	"avg_single_doc_ms",
	"avg_batched_doc_ms",
	"query_water_matches",
	"first_query_water_ms",
}

type Image struct {
	Name string
}

type im struct {
	Images []string
}

func main() {
	flag.Parse()
	lineptrs := [][]*Line{}
	var v []*Line
	if *configDir != "" {
		files, _ := ioutil.ReadDir(*configDir)
		for _, f := range files {
			fmt.Println(f.Name())
			if f.Name() == "." || f.Name() == ".." {
				continue
			}
			v = runConfig(*configDir+"/"+f.Name(), *target+"_"+f.Name(),
				*cpuprofile+"_"+f.Name())
			lineptrs = append(lineptrs, v)
			runtime.GC()
		}
	} else {
		v = runConfig(*config, *target, *cpuprofile)
		lineptrs = append(lineptrs, v)
	}
	if *doplot {
		fileNames := []string{}
		for k := 0; k < len(lineptrs[0]); k++ {
			kj := []*Line{}
			for _, o := range lineptrs {
				kj = append(kj, o[k])
			}
			err := doPlot(kj, typename[k], "docs", "time", typename[k]+".png")
			if err != nil {
				log.Fatalf("error plotting: %v", err)
			}
			fileNames = append(fileNames, typename[k]+".png")
		}
		output, err := os.OpenFile("output.html", os.O_CREATE|os.O_RDWR, 0666)
		m := &im{Images: fileNames}
		t, err := template.ParseFiles("result.tmpl")
		if err != nil {
			log.Fatalf("error parsing template: %v", err)
		}
		t.Execute(output, m)
	}
}

func runConfig(conf string, tar string, cpu string) []*Line {
	if cpu != "" {
		f, err := os.Create(cpu)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	start := time.Now()

	wikiReader, err := blevebench.NewWikiReader(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer wikiReader.Close()

	mapping := blevebench.BuildArticleMapping()
	benchConfig := blevebench.LoadConfigFile(conf)

	fmt.Printf("Using Index Type: %s\n", benchConfig.IndexType)
	fmt.Printf("Using KV store: %s\n", benchConfig.KVStore)
	fmt.Printf("Using KV config: %#v\n", benchConfig.KVConfig)
	index, err := bleve.NewUsing(tar, mapping, benchConfig.IndexType, benchConfig.KVStore, benchConfig.KVConfig)
	if err != nil {
		log.Fatal(err)
	}
	_, store, err := index.Advanced()
	if err != nil {
		log.Fatal(err)
	}

	itr := *count / (*level)
	lines := NewLines(itr, len(typename), conf, typename)
	tot := 0
	// print header
	fmt.Printf("elapsed,docs,avg_single_doc_ms,avg_batched_doc_ms,query_water_matches,first_query_water_ms,avg_repeated%d_query_water_ms", *qrepeat)
	printOtherHeader(store)
	fmt.Printf("\n")

	singleCount := 0
	var singleTime time.Duration
	batchCount := 0
	var batchTime time.Duration
	batch := index.NewBatch()
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
				batch = index.NewBatch()
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
			lines[0].Pt[tot].Y = avgSingleDocTime / float64(time.Millisecond)
			lines[0].Pt[tot].X = float64(i)
			lines[1].Pt[tot].Y = avgBatchDocTime / float64(time.Millisecond)
			lines[1].Pt[tot].X = float64(i)
			lines[2].Pt[tot].Y = firstQueryTime / float64(time.Millisecond)
			lines[2].Pt[tot].X = float64(i)
			lines[3].Pt[tot].Y = avgQueryTime / float64(time.Millisecond)
			lines[3].Pt[tot].X = float64(i)
			tot++

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
	return lines
}
