package main

import (
	"elsa-xml/pkg/extractor"
	"elsa-xml/pkg/validator"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	ErrorFormat    = "Error: %v\n"
	envVarTestData = "TEST_DATA_DIR"
)

func main() {
	testDataDir := os.Getenv(envVarTestData)
	if testDataDir == "" {
		exitWithError(errors.New("TEST_DATA_DIR environment variable not set"))
	}
	v, err := validator.NewValidator()
	if err != nil {
		exitWithError(err)
	}

	// get all xml files from testDataDir
	files, err := os.ReadDir(testDataDir)
	if err != nil {
		exitWithError(err)
	}

	// loop over files and validate each xml file
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fName := file.Name()
		sName := schemaName(fName)

		// load the file
		xmlFile, err := os.ReadFile(testDataDir + "/" + fName)
		if err != nil {
			exitWithError(err)
		}

		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("Processing %s\n", fName)

		// Execute schema validation
		err = v.Validate(xmlFile, sName)

		if err != nil {
			fmt.Printf("<<< Error >>>: %v\n", err)
			continue
		}
		fmt.Printf("File %s is valid\n", fName)

		// Extract data according to type
		result, err := extractor.Extract(xmlFile, msgName(fName))
		if err != nil {
			fmt.Printf("<<< Error >>>: %v\n", err)
			continue
		}
		if result != nil {
			printResult(result)
		} else {
			fmt.Println(">>> No Data! <<<")
		}

		fmt.Println(strings.Repeat("-", 80))
	}
}

func msgName(fileName string) string {
	parts := strings.Split(fileName, "_")
	if len(parts) < 2 {
		fmt.Printf("Error: %s is not a valid schema name\n", fileName)
		return ""
	}

	mName := parts[0]
	if parts[1] == "t2s" {
		mName = strings.ReplaceAll(parts[0], ".", "") + "plus"
	}
	return mName
}

func schemaName(fileName string) string {
	parts := strings.Split(fileName, "_")
	if len(parts) < 2 {
		fmt.Printf("Error: %s is not a valid schema name\n", fileName)
		return ""
	}

	sName := parts[0]
	if parts[1] == "t2s" {
		sName = "CST2SMsg"
	}
	return sName
}

func printResult(result *extractor.ExtractionResult) {
	keys := []string{extractor.TxIDKey,
		extractor.MovementTypeKey,
		extractor.PaymentTypeKey,
		extractor.MessageTypeKey,
		extractor.ReceivedFromKey,
		extractor.InstructingPartyKey}

	for _, k := range keys {
		fmt.Printf("%-16s: %s\n", k, result.Value(k))
	}
}

func exitWithError(err error) {
	fmt.Printf(ErrorFormat, err)
	os.Exit(1)
}
