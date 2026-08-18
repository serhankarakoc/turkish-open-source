package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ContributorsFile struct {
	Version      int           `json:"version"`
	GeneratedAt  string        `json:"generated_at,omitempty"`
	Contributors []Contributor `json:"contributors"`
}

type Contributor struct {
	Username           string   `json:"username"`
	HTMLURL            string   `json:"html_url"`
	Projects           []string `json:"projects"`
	TotalContributions int      `json:"total_contributions"`
}

func LoadDataset(path string) (Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EmptyDataset(time.Unix(0, 0).UTC()), nil
		}
		return Dataset{}, err
	}
	var ds Dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return Dataset{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if ds.Projects == nil {
		ds.Projects = []Project{}
	}
	if ds.Version == 0 {
		ds.Version = DatasetVersion
	}
	return ds, nil
}

func SaveDataset(path string, ds Dataset) error {
	if ds.Projects == nil {
		ds.Projects = []Project{}
	}
	SortProjects(ds.Projects)
	return writeJSON(path, ds)
}

func LoadContributors(path string) (ContributorsFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ContributorsFile{Version: DatasetVersion, Contributors: []Contributor{}}, nil
		}
		return ContributorsFile{}, err
	}
	var cf ContributorsFile
	if err := json.Unmarshal(raw, &cf); err != nil {
		return ContributorsFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if cf.Contributors == nil {
		cf.Contributors = []Contributor{}
	}
	return cf, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := marshalJSON(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func marshalJSON(v any) ([]byte, error) {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	buf = append(buf, '\n')
	return buf, nil
}

func MarshalJSON(v any) ([]byte, error) {
	return marshalJSON(v)
}
