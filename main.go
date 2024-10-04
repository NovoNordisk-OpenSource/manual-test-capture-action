package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TestResult struct {
	UUID        string       `json:"uuid"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Attachments []Attachment `json:"attachments"`
	Labels      []Label      `json:"labels"`
	Start       int64        `json:"start"`
	Stop        int64        `json:"stop"`
}

func main() {
	shutdownChan := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/generate", generateHandler(shutdownChan))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Goroutine to handle shutdown signal
	go func() {
		<-shutdownChan
		// Wait 5 seconds before shutting down
		time.Sleep(5 * time.Second)
		fmt.Println("Shutting down")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("Server Shutdown Failed:%+v", err)
		}
	}()

	fmt.Println("Server started at http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func generateHandler(shutdownChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the form data
		err := r.ParseMultipartForm(20 << 20) // Limit upload size to 20 MB
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}

		testName := r.FormValue("testName")
		testStatus := r.FormValue("testStatus")
		testTag := r.FormValue("testTag")
		featureName := r.FormValue("featureName")
		optionTag := r.FormValue("optionTag")

		// Handle file uploads
		attachmentsHeaders := r.MultipartForm.File["attachments[]"]

		var encodedAttachments []Attachment

		for _, fileHeader := range attachmentsHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Error opening attachment", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Read the file content
			fileBytes, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "Error reading attachment", http.StatusInternalServerError)
				return
			}

			// Encode the file content in base64
			encoded := base64.StdEncoding.EncodeToString(fileBytes)

			// Determine the MIME type
			var mimeType string
			if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".jpg") || strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".jpeg") {
				mimeType = "image/jpeg"
			} else if strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".png") {
				mimeType = "image/png"
			} else {
				// Unsupported file type
				continue
			}

			attachment := Attachment{
				Name:    fileHeader.Filename,
				Content: encoded,
				Type:    mimeType,
			}
			encodedAttachments = append(encodedAttachments, attachment)
		}

		// Create the labels
		labels := []Label{
			{
				Name:  "feature",
				Value: featureName,
			},
			{
				Name:  "tag",
				Value: testTag,
			},
			{
				Name:  "tag",
				Value: "manual",
			},
			{
				Name:  "tag",
				Value: optionTag,
			},
		}

		// Create the test result object
		currentTime := time.Now()
		testResult := TestResult{
			UUID:        testTag,
			Name:        testName,
			Status:      testStatus,
			Attachments: encodedAttachments,
			Labels:      labels,
			Start:       currentTime.UnixMilli(),
			Stop:        currentTime.UnixMilli(),
		}

		// Marshal the test result to JSON
		jsonData, err := json.MarshalIndent(testResult, "", "  ")
		if err != nil {
			http.Error(w, "Error generating JSON", http.StatusInternalServerError)
			return
		}

		// Create the output directory if it doesn't exist
		outputDir := "output"
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			err := os.Mkdir(outputDir, os.ModePerm)
			if err != nil {
				http.Error(w, "Error creating output directory", http.StatusInternalServerError)
				return
			}
		}

		// Create the filename
		filename := fmt.Sprintf("manual-test-%d-result.json", testResult.Stop)
		filepath := filepath.Join(outputDir, filename)

		// Save the JSON data to a file
		err = os.WriteFile(filepath, jsonData, 0644)
		if err != nil {
			http.Error(w, "Error saving file to disk", http.StatusInternalServerError)
			return
		}

		// Set the headers to force download with the same filename
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.Write(jsonData)

		// Signal the server to shut down
		go func() {
			shutdownChan <- struct{}{}
		}()
	}
}
