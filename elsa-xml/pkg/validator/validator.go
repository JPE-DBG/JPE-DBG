package validator

import (
	"errors"
	"fmt"
	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
	"os"
	"path/filepath"
	"strings"
)

type Validator struct {
	parsedSchemas map[string]*xsd.Schema
}

func NewValidator() (*Validator, error) {
	isoDir := os.Getenv(envVarISOSchemaDir)
	t2sDir := os.Getenv(envVarT2SSchemaDir)
	if isoDir == "" && t2sDir == "" {
		return nil, errors.New("no schema directories set")
	}

	v := Validator{
		parsedSchemas: make(map[string]*xsd.Schema),
	}

	if isoDir != "" {
		err := v.loadISOSchemas(isoDir)
		if err != nil {
			return nil, err
		}
	}

	if t2sDir != "" {
		err := v.loadT2SSchemas(t2sDir)
		if err != nil {
			return nil, err
		}
	}

	return &v, nil
}

func (v *Validator) Validate(xml []byte, schema string) error {
	doc, err := libxml2.Parse(xml)
	if err != nil {
		return err
	}
	defer doc.Free()

	if _, ok := v.parsedSchemas[schema]; !ok {
		return fmt.Errorf("schema %s not found", schema)
	}

	err = v.parsedSchemas[schema].Validate(doc)
	if err != nil {
		var svErr xsd.SchemaValidationError
		ok := errors.As(err, &svErr)
		if ok {
			// loop over the errors and print them (Currently not working, need to check with docu, may not work at all due to underlying library...)
			for _, e := range svErr.Errors() {
				fmt.Printf("Error: %s\n", e)
			}
		}
		return err
	}
	return nil
}

func (v *Validator) loadISOSchemas(isoDir string) error {
	isoFiles, err := os.ReadDir(isoDir)
	if err != nil {
		return err
	}
	for _, file := range isoFiles {
		if file.IsDir() {
			continue
		}
		schema, err := xsd.ParseFromFile(filepath.Join(isoDir, file.Name()))
		if err != nil {
			return err
		}
		v.parsedSchemas[strings.TrimSuffix(file.Name(), ".xsd")] = schema
	}
	return nil
}

func (v *Validator) loadT2SSchemas(t2sDir string) error {
	schema, err := xsd.ParseFromFile(filepath.Join(t2sDir, "CST2SMsg.valid.xsd"))
	if err != nil {
		return err
	}
	v.parsedSchemas["CST2SMsg"] = schema
	return nil
}
