package router

import (
	"groupie_tracker/controller"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// New crée et configure un *http.ServeMux pour l'application.
// Elle enregistre les handlers pour les routes HTML et l'API,
// et configure le serveur de fichiers statiques sous `/static/`.
func New() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", controller.HomeWithFavorites)
	mux.HandleFunc("/favorites", controller.Favorites)
	mux.HandleFunc("/about", controller.About)
	mux.HandleFunc("/contact", controller.Contact)
	mux.HandleFunc("/api/clubs", controller.SearchAndFilter)
	mux.HandleFunc("/add-favorite", controller.AddFavorite)
	mux.HandleFunc("/remove-favorite", controller.RemoveFavorite)
	mux.HandleFunc("/clear-favorites", controller.ClearFavorites)

	// Serve static files (images, css) from data/static under /static/
	staticDir := findStaticDir()
	if staticDir == "" {
		log.Printf("warning: data/static directory not found; static files won't be served")
	} else {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/static/", http.StripPrefix("/static/", fs))
		log.Printf("serving static files from %s at /static/", staticDir)
	}

	return mux
}

// findStaticDir recherche le répertoire `data/static` en remontant
// l'arborescence à partir du répertoire de travail courant (jusqu'à 6 niveaux).
// Elle retourne le chemin trouvé ou une chaîne vide si aucun répertoire n'a été trouvé.
func findStaticDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	// search up to 6 levels
	cur := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(cur, "data", "static")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			return candidate
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}
