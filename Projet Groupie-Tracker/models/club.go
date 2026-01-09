package models

import (
	"encoding/json"
	"fmt"
	"os"
)

// Club represents minimal club information used by the templates.
type Club struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	TLA       string `json:"tla,omitempty"`
	Website   string `json:"website,omitempty"`
	Founded   int    `json:"founded,omitempty"`
	Venue     string `json:"venue,omitempty"`
	CrestURL  string `json:"crestUrl,omitempty"`
}

// LoadClubsFromFile lit un fichier JSON contenant un tableau de clubs et
// renvoie la slice de `Club` correspondante.
// Pour être résiliente aux différents répertoires de travail, elle tente
// plusieurs chemins relatifs avant de renvoyer une erreur.
func LoadClubsFromFile(path string) ([]Club, error) {
	// Try a set of candidate paths so loading works regardless of working dir
	candidates := []string{
		path,
		"./" + path,
		"../" + path,
		"../../" + path,
		"data/clubs.json",
	}
	var b []byte
	var err error
	var found string
	for _, p := range candidates {
		b, err = os.ReadFile(p)
		if err == nil {
			found = p
			break
		}
	}
	if found == "" {
		return nil, fmt.Errorf("clubs JSON not found; tried: %v; last error: %w", candidates, err)
	}
	var clubs []Club
	if err := json.Unmarshal(b, &clubs); err != nil {
		return nil, err
	}
	return clubs, nil
}
