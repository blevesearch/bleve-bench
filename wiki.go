package blevebench

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Article struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type WikiReader struct {
	file   *os.File
	reader *bufio.Reader
}

func NewWikiReader(path string) (*WikiReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// read off the garbage header
	br := bufio.NewReader(f)
	br.ReadString('\t')
	return &WikiReader{
		file:   f,
		reader: br,
	}, nil
}

func (w *WikiReader) Next() (*Article, error) {
	line, err := w.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	parts := strings.Split(line, "\t")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid line: %s", line)
	}
	a := Article{
		Title: parts[0],
		Text:  parts[2],
	}
	return &a, nil
}

func (w *WikiReader) Close() error {
	return w.file.Close()
}
