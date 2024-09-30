package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
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
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/generate", generateHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
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
	attachments := r.MultipartForm.File["attachments[]"]

	var encodedAttachments []Attachment

	for _, fileHeader := range attachments {
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
	testResult := TestResult{
		UUID:        testTag,
		Name:        testName,
		Status:      testStatus,
		Attachments: encodedAttachments,
		Labels:      labels,
		Start:       time.Now().UnixMilli(),
		Stop:        time.Now().UnixMilli(),
	}

	// Marshal the test result to JSON
	jsonData, err := json.MarshalIndent(testResult, "", "  ")
	if err != nil {
		http.Error(w, "Error generating JSON", http.StatusInternalServerError)
		return
	}

	// Set the headers to force download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"test-result.json\"")
	w.Write(jsonData)
}
