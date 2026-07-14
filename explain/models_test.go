package explain

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ardanlabs/kronk/sdk/tools/models"
)

type fakeModelFileLister struct {
	files []models.File
	err   error
}

func (fake fakeModelFileLister) Files() ([]models.File, error) {
	return fake.files, fake.err
}

func TestInstalledVisionModels(t *testing.T) {
	lister := fakeModelFileLister{files: []models.File{
		{ID: "vision-b", Validated: true, HasProjection: true},
		{ID: "text", Validated: true},
		{ID: "unvalidated-vision", HasProjection: true},
		{ID: "vision-a", Validated: true, HasProjection: true},
	}}

	got, err := installedVisionModels(lister)
	if err != nil {
		t.Fatalf("installedVisionModels: %v", err)
	}
	want := []string{"vision-a", "vision-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
}

func TestInstalledVisionModelsError(t *testing.T) {
	want := errors.New("list models")
	_, got := installedVisionModels(fakeModelFileLister{err: want})
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}
