package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/BurntSushi/toml"
	"github.com/osbuild/image-builder/internal/oscap"
)

func genXMLTailoringFile(tailoring *string, datastream string) (*os.File, error) {
	if tailoring == nil || *tailoring == "" {
		// we don't need to process this any further,
		// since the xml file will just end up blank
		// and would cause issues later down the line
		return nil, nil
	}

	jsonFile, err := os.CreateTemp("", "tailoring.json")
	if err != nil {
		return nil, fmt.Errorf("Error creating temp json file: %w", err)
	}
	defer os.Remove(jsonFile.Name())

	_, err = jsonFile.Write([]byte(*tailoring))
	if err != nil {
		return nil, fmt.Errorf("Error writing json customizations to temp file: %w", err)
	}

	xmlFile, err := os.CreateTemp("", "tailoring.xml")
	if err != nil {
		return nil, fmt.Errorf("Error creating temp xml file: %w", err)
	}

	// TODO: json schema validation
	// we could potentially validate the `json` input
	// here against:
	// https://github.com/ComplianceAsCode/schemas/blob/b91c8e196a8cc515e0cc7f10b2c5a02b4179c0e5/tailoring/schema.json
	// Alternatively, we could just fetch the `xml` blob from the compliance service and
	// skip this step altogether

	// The oscap blueprint generation tool
	// doesn't accept `json` as input, so we
	// need to convert it to `xml`
	cmd := exec.Command(
		"autotailor",
		"-j", jsonFile.Name(),
		"-o", xmlFile.Name(),
		datastream,
	)

	if err := cmd.Run(); err != nil {
		os.Remove(xmlFile.Name())
		return nil, fmt.Errorf("Error executing blueprint generation: %w", err)
	}

	return xmlFile, nil
}

func genTOMLBlueprint(profile string, datastream string, file *os.File) ([]byte, error) {
	var cmd *exec.Cmd
	if file != nil {
		cmd = exec.Command("oscap",
			"xccdf",
			"generate",
			"fix",
			"--profile",
			string(profile),
			"--tailoring-file",
			file.Name(),
			"--fix-type",
			"blueprint",
			datastream,
		)
	} else {
		cmd = exec.Command("oscap",
			"xccdf",
			"generate",
			"fix",
			"--profile",
			string(profile),
			"--fix-type",
			"blueprint",
			datastream,
		)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Error generating toml blueprint: %w", err)
	}

	return output, nil
}

func processRequest(profile string, datastream string, tailoring *string) ([]byte, error) {
	// the generated blueprint doesn't contain the profile
	// description, so we have to run the oscap tool to get
	// this information
	description := oscap.GetProfileDescription(profile, datastream)

	file, err := genXMLTailoringFile(tailoring, datastream)
	if err != nil {
		return nil, err
	}
	defer func() {
		if file != nil {
			os.Remove(file.Name())
		}
	}()

	rawBp, err := genTOMLBlueprint(profile, datastream, file)
	if err != nil {
		return nil, err
	}

	var bp *oscap.Blueprint
	err = toml.Unmarshal(rawBp, &bp)
	if err != nil {
		return nil, err
	}

	customizations, err := oscap.BlueprintToCustomizations(profile, description, *bp)
	if err != nil {
		return nil, err
	}

	jCustomizations, err := json.Marshal(customizations)
	if err != nil {
		return nil, err
	}

	return jCustomizations, nil
}
