package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/gherkin-go/v19"
	messages "github.com/cucumber/messages-go/v16"
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

type Scenario struct {
	Name         string
	Description  string
	Steps        []string
	OptionTag    string
	FeatureTag   string
	FeatureName  string
	ScenarioText string
}

var (
	scenarios                   []Scenario
	currentScenarioIndex        int
	shutdownOnce                sync.Once
	optionTagToProcess          string
	optionTagToProcessLowercase string
	webpageTitle                string
	environment                 string
)

func main() {
	// Parse command-line arguments
	var pvFlag, ivFlag, ppvFlag, pivFlag bool
	var featuresDirFlag string
	var environmentFlag string
	flag.BoolVar(&pvFlag, "pv", false, "Process @manual scenarios with @PV tag")
	flag.BoolVar(&ivFlag, "iv", false, "Process @manual scenarios with @IV tag")
	flag.BoolVar(&ppvFlag, "ppv", false, "Process @manual scenarios with @pPV tag")
	flag.BoolVar(&pivFlag, "piv", false, "Process @manual scenarios with @pIV tag")
	flag.StringVar(&featuresDirFlag, "features-dir", "requirements", "Relative path to directory containing .feature files")
	flag.StringVar(&environmentFlag, "environment", "", "Environment the tests are executed in, options: [validation|production]")
	flag.Parse()

	// Propagate environment to the global variable
	environment = environmentFlag

	// Determine which option tag to process
	if pvFlag {
		optionTagToProcess = "@PV"
		optionTagToProcessLowercase = "pv"
		webpageTitle = "Test Scenarios (PV)"
	} else if ivFlag {
		optionTagToProcess = "@IV"
		optionTagToProcessLowercase = "iv"
		webpageTitle = "Test Scenarios (IV)"
	} else if ppvFlag {
		optionTagToProcess = "@pPV"
		optionTagToProcessLowercase = "ppv"
		webpageTitle = "Test Scenarios (pPV)"
	} else if pivFlag {
		optionTagToProcess = "@pIV"
		optionTagToProcessLowercase = "piv"
		webpageTitle = "Test Scenarios (pIV)"
	} else {
		log.Fatal("Please specify one of --pv, --iv, --ppv, or --piv")
	}

	// Load scenarios from feature files
	err := loadScenarios(featuresDirFlag)
	if err != nil {
		log.Fatalf("Error loading scenarios: %v", err)
	}

	shutdownChan := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler(shutdownChan))
	mux.HandleFunc("/generate", generateHandler)
	mux.HandleFunc("/download", downloadHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

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

func loadScenarios(featuresDir string) error {
	err := filepath.WalkDir(featuresDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".feature") {
			err := parseFeatureFile(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func parseFeatureFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	gherkinDocument, err := gherkin.ParseGherkinDocument(strings.NewReader(string(content)), (&messages.Incrementing{}).NewId)
	if err != nil {
		return err
	}

	feature := gherkinDocument.Feature
	if feature == nil {
		return fmt.Errorf("No feature found in %s", path)
	}

	// Get the last tag of the Feature
	var featureTag string
	if len(feature.Tags) > 0 {
		lastTag := feature.Tags[len(feature.Tags)-1]
		featureTag = lastTag.Name
	}

	// Get the text after "Feature:" for the feature name
	featureName := feature.Name

	for _, child := range feature.Children {
		if child.Scenario != nil {
			scenario := child.Scenario

			// Only process scenarios tagged with @manual
			hasManualTag := false
			var optionTags []string

			for _, tag := range scenario.Tags {
				if tag.Name == "@manual" {
					hasManualTag = true
				}
				if tag.Name == "@PV" || tag.Name == "@IV" || tag.Name == "@pPV" || tag.Name == "@pIV" {
					optionTags = append(optionTags, tag.Name)
				}
			}
			if !hasManualTag {
				continue
			}

			// Only process scenarios that have the specified option tag among their tags
			if !contains(optionTags, optionTagToProcess) {
				continue
			}

			sc := Scenario{
				Name:        scenario.Name,
				Steps:       []string{},
				OptionTag:   optionTagToProcess,
				FeatureTag:  featureTag,
				FeatureName: featureName,
			}

			// Collect steps
			for _, step := range scenario.Steps {
				sc.Steps = append(sc.Steps, step.Keyword+step.Text)
			}

			// Collect scenario text
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("%s: %s\n", scenario.Keyword, scenario.Name))
			for _, step := range scenario.Steps {
				sb.WriteString(fmt.Sprintf("  %s %s\n", step.Keyword, step.Text))
			}
			// Handle Examples if Scenario Outline
			if len(scenario.Examples) > 0 {
				sb.WriteString("\nExamples:\n")
				for _, example := range scenario.Examples {
					// Write table header
					sb.WriteString("  |")
					for _, cell := range example.TableHeader.Cells {
						sb.WriteString(fmt.Sprintf(" %s |", cell.Value))
					}
					sb.WriteString("\n")
					// Write table rows
					for _, row := range example.TableBody {
						sb.WriteString("  |")
						for _, cell := range row.Cells {
							sb.WriteString(fmt.Sprintf(" %s |", cell.Value))
						}
						sb.WriteString("\n")
					}
				}
			}

			sc.ScenarioText = sb.String()

			scenarios = append(scenarios, sc)
		}
	}
	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type TemplateData struct {
	CurrentScenario       Scenario
	Scenarios             []Scenario
	CurrentIndex          int
	AllScenariosProcessed bool
	WebpageTitle          string
}

func indexHandler(shutdownChan chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		data := TemplateData{
			Scenarios:             scenarios,
			CurrentIndex:          currentScenarioIndex,
			AllScenariosProcessed: currentScenarioIndex >= len(scenarios),
			WebpageTitle:          webpageTitle,
		}

		if !data.AllScenariosProcessed {
			data.CurrentScenario = scenarios[currentScenarioIndex]
		}

		tmpl.Execute(w, data)

		if data.AllScenariosProcessed {
			// All scenarios are processed, signal shutdown
			shutdownOnce.Do(func() {
				go func() {
					shutdownChan <- struct{}{}
				}()
			})
		}
	}
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
	comments := r.FormValue("comments")
	startTimestampStr := r.FormValue("startTimestamp")

	var startTimestamp int64
	if startTimestampStr != "" {
		startTimestamp, err = strconv.ParseInt(startTimestampStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid start timestamp", http.StatusBadRequest)
			return
		}
	} else {
		// If for some reason the start timestamp is missing, use current time
		startTimestamp = time.Now().UnixMilli()
	}

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
		{
			Name:  "comments",
			Value: comments,
		},
	}

	// Create the test result object
	testResult := TestResult{
		UUID:        testTag,
		Name:        testName,
		Status:      testStatus,
		Attachments: encodedAttachments,
		Labels:      labels,
		Start:       startTimestamp,
		Stop:        time.Now().UnixMilli(),
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
	filename := fmt.Sprintf("manual-test-%s-%s-%d-result.json", environment, optionTagToProcessLowercase, testResult.Stop)
	filepath := filepath.Join(outputDir, filename)

	// Save the JSON data to a file
	err = os.WriteFile(filepath, jsonData, 0644)
	if err != nil {
		http.Error(w, "Error saving file to disk", http.StatusInternalServerError)
		return
	}

	// Generate the URL for downloading the file
	downloadURL := fmt.Sprintf("/download?filename=%s", url.QueryEscape(filename))

	// Write the HTML response with JavaScript to trigger the download and reload the page
	// Apply the PicoCSS stylesheet to this page
	htmlResponse := fmt.Sprintf(`
<html lang="en" data-theme="dark">
<head>
	<meta charset="UTF-8">
	<title>Processing</title>
	<link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
	<script type="text/javascript">
		window.onload = function() {
			window.location.href = "%s";
			setTimeout(function() {
				window.location.href = "/";
			}, 2000);
		};
	</script>
</head>
<body>
	<main class="container">
		<h1>Processing...</h1>
	</main>
</body>
</html>
`, downloadURL)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlResponse))

	// Increment the current scenario index
	currentScenarioIndex++
	if currentScenarioIndex >= len(scenarios) {
		currentScenarioIndex = len(scenarios)
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Filename not specified", http.StatusBadRequest)
		return
	}

	// Ensure filename is safe
	if !strings.HasPrefix(filename, "manual-test-") || !strings.HasSuffix(filename, "-result.json") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join("output", filename)
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Serve the file
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.ServeFile(w, r, filepath)
}
