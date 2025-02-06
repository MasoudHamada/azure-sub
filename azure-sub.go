package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Configuration constants
const (
	wordlistPath = "/usr/share/seclists/Discovery/DNS/subdomains-top1million-110000.txt"
	numWorkers   = 50
	timeout      = 5 * time.Second
)

// Base domains and permutation functions
var baseURLs = []string{
	"azurewebsites.net",
	"blob.core.windows.net",
	"queue.core.windows.net",
	"file.core.windows.net",
	"table.core.windows.net",
	"scm.azurewebsites.net",
}

// Permutation functions define the four formats.
var permutations = []func(word, company string) string{
	// {word}-{company}
	func(word, company string) string { return word + "-" + company },
	// {company}-{word}
	func(word, company string) string { return company + "-" + word },
	// {word}{company}
	func(word, company string) string { return word + company },
	// {company}{word}
	func(word, company string) string { return company + word },
}

// checkSubdomain sends an HTTPS GET request to the given subdomain.
// If the response status is 200 or 302, it sends the valid subdomain (with status) to validChan.
func checkSubdomain(client *http.Client, subdomain string, validChan chan<- string) {
	url := "https://" + subdomain
	resp, err := client.Get(url)
	if err != nil {
		return // Ignore errors (e.g. timeout, no route, TLS errors)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
		// Format the valid output (you may adjust to your liking)
		validResult := fmt.Sprintf("[VALID] %s (%d)", subdomain, resp.StatusCode)
		validChan <- validResult
	}
}

func main() {
	// Check for the required argument (company name)
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <company_name>\n", os.Args[0])
		os.Exit(1)
	}
	company := os.Args[1]

	// Read the wordlist file
	words, err := readWordlist(wordlistPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading wordlist: %v\n", err)
		os.Exit(1)
	}

	// Channels for job distribution and valid subdomain collection.
	jobs := make(chan string, 1000)
	validChan := make(chan string, 100)

	// HTTP client with timeout.
	client := &http.Client{
		Timeout: timeout,
	}

	// WaitGroup for worker goroutines.
	var wg sync.WaitGroup

	// Launch a fixed number of workers.
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for subdomain := range jobs {
				checkSubdomain(client, subdomain, validChan)
			}
		}()
	}

	// Start a writer goroutine that writes valid subdomains to file and stdout.
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		writeValidResults(validChan)
	}()

	// Generate subdomains and send them to the jobs channel.
	count := 0
	for _, base := range baseURLs {
		for _, word := range words {
			for _, permute := range permutations {
				subdomain := permute(word, company) + "." + base
				jobs <- subdomain
				count++
			}
		}
	}
	fmt.Printf("[*] Testing %d subdomains...\n", count)
	close(jobs) // Signal workers that no more jobs will be sent.

	// Wait for all workers to finish processing.
	wg.Wait()
	close(validChan) // Signal writer that no more valid results will come.
	writerWg.Wait()  // Wait for writer to finish.
}

// readWordlist opens the file at path and returns a slice of words.
func readWordlist(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// You might wish to add filtering (e.g., ignore empty lines)
		if line != "" {
			words = append(words, line)
		}
	}
	return words, scanner.Err()
}

// writeValidResults reads from the validChan channel and writes each valid subdomain
// to stdout and appends it to valid_subdomains.txt.
func writeValidResults(validChan <-chan string) {
	// Open the file in append mode (or create it if it doesn't exist).
	f, err := os.OpenFile("valid_subdomains.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening valid_subdomains.txt: %v\n", err)
		return
	}
	defer f.Close()

	for result := range validChan {
		fmt.Println(result)
		if _, err := f.WriteString(result + "\n"); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to valid_subdomains.txt: %v\n", err)
		}
	}
}
