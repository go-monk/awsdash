package explain

import (
	"sort"

	"github.com/ardanlabs/kronk/sdk/tools/models"
)

type modelFileLister interface {
	Files() ([]models.File, error)
}

// InstalledVisionModels returns locally installed and validated models that
// include a multimodal projection file.
func InstalledVisionModels() ([]string, error) {
	manager, err := models.New()
	if err != nil {
		return nil, err
	}
	return installedVisionModels(manager)
}

func installedVisionModels(lister modelFileLister) ([]string, error) {
	files, err := lister.Files()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(files))
	for _, file := range files {
		if file.Validated && file.HasProjection {
			ids = append(ids, file.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
