// +build ignore

package main

import (
	"compress/bzip2"
	"flag"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/dustin/go-wikiparse"
)

var numWorkers = flag.Int("numWorkers", 8, "Number of page workers")

var benchTimeStampFormat = "02-Jan-2006 15:04:05.000"

func doHeader(f *os.File) {
	f.WriteString("FIELDS_HEADER_INDICATOR###\tdoctitle\tdocdate\tbody\n")
}

func doPage(of *os.File, cf *os.File, p *wikiparse.Page) {
	f := of
	if strings.HasPrefix(p.Title, "Category:") {
		f = cf
	}

	startTextEnd := len(p.Revisions[0].Text)
	if startTextEnd > 9 {
		startTextEnd = 9
	}
	startText := strings.ToLower(p.Revisions[0].Text[0:startTextEnd])
	if startText == "#redirect" {
		return
	}

	f.WriteString(p.Title)
	f.WriteString("\t")
	t, err := time.Parse(time.RFC3339, p.Revisions[0].Timestamp)
	if err != nil {
		log.Printf("error parsing time: %v", err)
	}
	f.WriteString(t.Format(benchTimeStampFormat[0:3]))
	f.WriteString(strings.ToUpper(t.Format(benchTimeStampFormat[3:6])))
	f.WriteString(t.Format(benchTimeStampFormat[6:]))
	f.WriteString("\t")
	textTrim := strings.Trim(p.Revisions[0].Text, "\n\t ")
	textWithoutNewlines := strings.Replace(textTrim, "\n", " ", -1)
	textWithoutNewlinesOrTabs := strings.Replace(textWithoutNewlines, "\t", " ", -1)
	f.WriteString(textWithoutNewlinesOrTabs)
	f.WriteString("\n")
}

func main() {
	procs := flag.Int("cpus", runtime.NumCPU(), "Number of CPUS to use")
	flag.Parse()

	var input io.Reader
	input, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error opening input file: %v", err)
	}

	if strings.HasSuffix(flag.Arg(0), ".bz2") {
		input = bzip2.NewReader(input)
	}

	output, err := os.OpenFile(flag.Arg(1), os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error opening output file: %v", err)
	}
	defer output.Close()
	doHeader(output)

	catoutput, err := os.OpenFile(flag.Arg(2), os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Error opening categories output file: %v", err)
	}
	defer catoutput.Close()
	doHeader(catoutput)

	runtime.GOMAXPROCS(*procs)

	p, err := wikiparse.NewParser(input)
	if err != nil {
		log.Fatalf("Error initializing parser: %v", err)
	}

	pages := int64(0)
	start := time.Now()
	prev := start
	reportfreq := int64(1000)
	for err == nil {
		var page *wikiparse.Page
		page, err = p.Next()
		if err == nil {
			doPage(output, catoutput, page)
		}

		pages++
		if pages%reportfreq == 0 {
			now := time.Now()
			d := now.Sub(prev)
			log.Printf("Processed %s pages total (%.2f/s)",
				humanize.Comma(pages), float64(reportfreq)/d.Seconds())
			prev = now
		}
	}
	log.Printf("Ended with err after %v:  %v after %s pages",
		time.Now().Sub(start), err, humanize.Comma(pages))
}
