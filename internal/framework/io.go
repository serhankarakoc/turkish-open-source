package framework

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func LoadDataset(path string) (Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EmptyDataset(time.Time{}), nil
		}
		return Dataset{}, err
	}
	var ds Dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return Dataset{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if ds.Frameworks == nil {
		ds.Frameworks = []Framework{}
	}
	if ds.Version == 0 {
		ds.Version = DatasetVersion
	}
	return ds, nil
}

func SaveDataset(path string, ds Dataset) error {
	if ds.Frameworks == nil {
		ds.Frameworks = []Framework{}
	}
	SortFrameworks(ds.Frameworks)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
