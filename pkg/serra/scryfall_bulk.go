package serra

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type BulkIndex struct {
	Object  string `json:"object"`
	HasMore bool   `json:"has_more"`
	Data    []struct {
		Object           string    `json:"object"`
		ID               string    `json:"id"`
		Type             string    `json:"type"`
		UpdatedAt        time.Time `json:"updated_at"`
		URI              string    `json:"uri"`
		Name             string    `json:"name"`
		Description      string    `json:"description"`
		JsonlDownloadURI string    `json:"jsonl_download_uri"`
		CompressedSize   int       `json:"compressed_size"`
	} `json:"data"`
}

func fetchBulkDownloadURL() (string, error) {
	downloadURL := ""

	// Make an HTTP GET request
	resp, err := queryScryfall("https://api.scryfall.com/bulk-data")
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	// Unmarshal the JSON response
	var bulkData BulkIndex
	if err := json.Unmarshal(body, &bulkData); err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	// Find and print the unique cards URL
	for _, item := range bulkData.Data {
		if item.Type == "default_cards" {
			downloadURL = item.JsonlDownloadURI
		}
	}

	return downloadURL, nil
}

func downloadBulkData(downloadURL string) (string, error) {

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "download")
	if err != nil {
		log.Fatalf("Error creating temporary directory: %v", err)
	}
	// defer os.RemoveAll(tempDir) // Clean up the directory when done

	// Create a temporary file in the temporary directory
	tempFile, err := os.CreateTemp(tempDir, "downloaded-*.json") // Adjust the extension if necessary
	if err != nil {
		log.Fatalf("Error creating temporary file: %v", err)
	}
	// defer tempFile.Close() // Ensure we close the file when we're done

	// Download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		log.Fatalf("Error downloading file: %v", err)
	}
	defer resp.Body.Close() // Make sure to close the response body

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d", resp.StatusCode)
	}

	// Copy the response body to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		log.Fatalf("Error saving file: %v", err)
	}

	return tempFile.Name(), nil
}

func loadBulkFile(gzipFilePath string) ([]Card, error) {
	// Open gzip file
	f, err := os.Open(gzipFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		f.Close()
		_ = os.Remove(gzipFilePath)
	}()

	// Wrap with gzip reader
	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	// Stream JSONL: one JSON object per line
	scanner := bufio.NewScanner(gr)
	// If your lines can be large, increase the max token size:
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var cards []Card
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var c Card
		if err := json.Unmarshal(line, &c); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cards, nil
}

func getCardFromBulk(cards []Card, setName, collectorNumber string) (*Card, error) {
	var foundCard Card
	for _, v := range cards {
		if v.CollectorNumber == collectorNumber && v.Set == setName {
			foundCard = v
			return &foundCard, nil
		}
	}
	return &Card{}, fmt.Errorf("Card %s/%s not found in bulk data", setName, collectorNumber)
}
