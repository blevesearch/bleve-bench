package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	testdir := flag.String("testdir", "", "testdir root")
	column := flag.String("column", "", "column to aggregate")
	configs := flag.String("configs", "", "configs to consider")
	source := flag.String("source", "", "source csv to operate on")
	statFilename := flag.String("filename", "", "stats filename")
	label := flag.String("label", "", "test label")
	flag.Parse()

	if *column == "" {
		log.Fatalf("must specify column to aggregate")
	}

	configNames := strings.Split(*configs, ",")
	if len(configNames) < 1 {
		log.Fatalf("must specify at least one config")
	}

	values := make(map[string]string)

	for _, config := range configNames {
		log.Printf("working on config: %s", config)
		files, err := filesForConfig(*testdir, config, *label, *statFilename)
		if err != nil {
			log.Fatalf("error finding output files: %v", err)
		}
		if len(files) < 1 {
			log.Fatalf("need at least one file to aggregate")
		}
		avg, stddev, err := processFiles(*column, files)
		if err != nil {
			log.Fatal(err)
		}
		values[config] = fmt.Sprintf("%f,%f", avg, stddev)
	}

	err := processSource(*source, *label, values)
	if err != nil {
		log.Fatalf("error processing source: %v", err)
	}

}

func filesForConfig(testdir, config, label, statsFilename string) ([]string, error) {
	pattern := testdir + string(os.PathSeparator) +
		label + string(os.PathSeparator) +
		config + string(os.PathSeparator) +
		"*" + string(os.PathSeparator) +
		statsFilename

	log.Printf("pattern is %s", pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func processSource(source, label string, values map[string]string) error {
	log.Printf("processing source")
	r, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("error opening source '%s': %v", source, err)
	}
	csvReader := csv.NewReader(r)
	headers, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("error reading header of source: %v", err)
	}
	err = r.Close()
	if err != nil {
		return fmt.Errorf("error closing source file: %v", err)
	}

	line := ""
	for i, header := range headers {
		if i == 0 {
			line += label
		} else {
			line += ","
			if val, ok := values[header]; ok {
				line += fmt.Sprintf("%s", val)
			}
		}
	}
	line += "\n"

	file, err := os.OpenFile(source, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return fmt.Errorf("error opening file for append: %v", err)
	}
	file.WriteString(line)
	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing source file after append: %v", err)
	}

	return nil
}

func processFiles(column string, files []string) (float64, float64, error) {
	values := make([]float64, len(files))

	for i, arg := range files {
		r, err := os.Open(arg)
		if err != nil {
			return 0.0, 0.0, fmt.Errorf("error opening file '%s': %v", arg, err)
		}
		csvReader := csv.NewReader(r)
		allRows, err := csvReader.ReadAll()
		if err != nil {
			return 0.0, 0.0, fmt.Errorf("error reading csv: %v", err)
		}
		if len(allRows) < 2 {
			return 0.0, 0.0, fmt.Errorf("csv file must contain header row and at least 1 data row")
		}
		headerRow := allRows[0]
		workcol := -1
		for col, headerName := range headerRow {
			if headerName == column {
				workcol = col
			}
		}
		if workcol < 0 {
			return 0.0, 0.0, fmt.Errorf("unable to find header column '%s'", column)
		}
		lastRow := allRows[len(allRows)-1]
		lastVal := lastRow[workcol]
		val, err := strconv.ParseFloat(lastVal, 64)
		if err != nil {

			return 0.0, 0.0, fmt.Errorf("unable to parse value '%s' as float: %v", lastVal, err)
		}
		values[i] = val
	}
	avg := average(values)
	return avg, stddev(values, avg), nil
}

func average(inputs []float64) float64 {
	sum := 0.0
	for _, input := range inputs {
		sum += input
	}
	return sum / float64(len(inputs))
}

func stddev(inputs []float64, population_avg float64) float64 {
	deviations := make([]float64, len(inputs))
	for i, input := range inputs {
		diff := input - population_avg
		diff = diff * diff
		deviations[i] = diff
	}
	variance := average(deviations)
	return math.Sqrt(variance)
}
