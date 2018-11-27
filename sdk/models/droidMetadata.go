package models

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// DroidMetadata defines the data model used in metadata.yml
type DroidMetadata struct {
	Kind         string                 `yaml:"kind"`
	Version      string                 `yaml:"version"`
	Product      string                 `yaml:"product"`
	Storage      bool                   `yaml:"storage"`
	Environments []DroidMetadataEnvDef  `yaml:"environments"`
	SecretFiles  []DroidMetadataFileDef `yaml:"secretFiles"`
}

// DroidMetadataFileDef defines the data model of the secret files definition in metadata.yml
type DroidMetadataFileDef struct {
	Path      string `yaml:"path"`
	SecretKey string `yaml:"secretKey"`
}

// DroidMetadataEnvDef defines the data model of the environment variable definition in metadata.yml
type DroidMetadataEnvDef struct {
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// ReadDroidMetadata reads the droid metadata from the metadata.yml file.
func ReadDroidMetadata(filePath string) (*DroidMetadata, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var metadata DroidMetadata
	err = yaml.Unmarshal(content, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
