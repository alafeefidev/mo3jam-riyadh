package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var( 
	lettersNums = map[int]string{
		2: "ء",
		3: "ب",
		4: "ت",
		5: "ث",
		6: "ج",
		7: "ح",
		8: "خ",
		9: "د",
		10: "ذ",
		11: "ر",
		12: "ز",
		13: "س",
		14: "ش",
		15: "ص",
		16: "ض",
		17: "ط",
		18: "ظ",
		19: "ع",
		20: "غ",
		21: "ف",
		22: "ق",
		23: "ك",
		24: "ل",
		25: "م",
		26: "ن",
		27: "هـ",
		28: "و",
		29: "ي",
	}

	Reset  = "\033[0m"
	Green  = "\033[32m"
	Purple = "\033[35m"
	Red    = "\033[31m"
	Cyan   = "\033[36m"
)
const (
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
	WordsURL = "https://siwar.ksaa.gov.sa/api/riyadh/Search/GetLetterWords?id=%d"
	ExpressionsURL = "https://siwar.ksaa.gov.sa/api/riyadh/Search/GetLetterExpressions?id=%d"
)

func main() {
	log.Println(colorize(Cyan, "Starting..."))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := orchestrator(ctx, "words_list", WordsURL, lettersNums); err != nil {
		log.Fatal(err)
	}

	if err := orchestrator(ctx, "expressions_list", ExpressionsURL, lettersNums); err != nil {
		log.Fatal(err)
	}

	log.Println(colorize(Cyan, "Done"))
}

func writer(path string, words []string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error in writing/opening file: %w", err)
	}
	defer func()  {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}()

	_, err = file.WriteString(strings.Join(words, "\n"))
	if err != nil {
		return fmt.Errorf("error in writing content of file: %w", err)
	}
	return nil
}

func orchestrator(ctx context.Context, dir, urlFormat string, mapping map[int]string) error {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

loop:
	for num, letter := range mapping {
		log.Printf(colorize(Purple, "[%s] Fetching for letter: %s"), dir, letter)
		url := fmt.Sprintf(urlFormat, num)
		select {
		case <- ctx2.Done():
			break loop
		default:
		}
	
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf(colorize(Purple, "[%s] Requesting url: %s"), dir, url)
			result, err := requester(ctx, url)
			if err != nil {
				log.Printf(colorize(Red, "[%s] ERROR fetching url: %s"), dir, url)
				select {
				case errCh <- err:
					cancel()
				default:
				}
			}

			log.Printf(colorize(Purple, "[%s] Writing: %s.tx"), dir, letter)
			path := filepath.Join(dir, fmt.Sprintf("%s.txt", letter))
			err = writer(path, result)
			if err != nil {
				log.Printf(colorize(Red, "[%s] ERROR writing: %s.txt"), dir, letter)
				select {
				case errCh <- err:
					cancel()
				default:
				}
			}
			log.Printf(colorize(Purple, "[%s] Done: %s"), dir, letter)
		}()
	}

	wg.Wait()
	close(errCh)

	return <-errCh
}

func requester(ctx context.Context, url string) ([]string, error) {
	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error in building the request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error in getting response from server: %w", err)
	}
	defer func()  {
		err := resp.Body.Close()
		if err != nil {
			panic(err)
		}
	}()
	log.Printf(colorize(Green, "GET %s -> %d"), url, resp.StatusCode)

	var result []string
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error in marshaling the JSON data from server: %w", err)
	}

	return result, nil
}

func colorize(color, text string) string {
        return color + text + Reset
}