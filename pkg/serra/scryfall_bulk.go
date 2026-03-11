package serra

import (
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
		Object          string    `json:"object"`
		ID              string    `json:"id"`
		Type            string    `json:"type"`
		UpdatedAt       time.Time `json:"updated_at"`
		URI             string    `json:"uri"`
		Name            string    `json:"name"`
		Description     string    `json:"description"`
		Size            int       `json:"size"`
		DownloadURI     string    `json:"download_uri"`
		ContentType     string    `json:"content_type"`
		ContentEncoding string    `json:"content_encoding"`
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
			downloadURL = item.DownloadURI
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

func loadBulkFile(bulkFilePath string) ([]Card, error) {

	var cards []Card
	fileBytes, _ := os.ReadFile(bulkFilePath)
	defer os.Remove(bulkFilePath)

	err := json.Unmarshal(fileBytes, &cards)
	if err != nil {
		fmt.Println("Error unmarshalling bulk file:", err)
		return cards, nil
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
