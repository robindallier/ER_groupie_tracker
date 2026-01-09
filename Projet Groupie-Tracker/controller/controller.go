package controller

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"groupie_tracker/models"
)

type PageData struct {
	Title   string
	Message string
	Clubs   []models.Club
}

type FilterResponse struct {
	Clubs      []models.Club `json:"clubs"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// toJSON convertit une valeur Go en JSON sûr pour les templates.
// Elle renvoie un `template.JS` contenant l'encodage JSON ou `null`
// en cas d'erreur d'encodage, afin d'éviter un plantage côté template.
func toJSON(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

// renderTemplate localise et exécute un fichier de template HTML.
// Elle cherche le template dans plusieurs chemins relatifs, prépare
// la fonction `toJSON` pour les templates et écrit la sortie dans `w`.
// En cas d'erreur de parsing ou d'exécution, elle logge et renvoie
// une erreur HTTP 500 au client.
func renderTemplate(w http.ResponseWriter, filename string, data interface{}) {

	candidates := []string{
		"template/" + filename,
		"./template/" + filename,
		"../template/" + filename,
		"../../template/" + filename,
	}
	var path string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		// none found
		msg := "template file missing; tried: " + candidates[0]
		for _, c := range candidates[1:] {
			msg += ", " + c
		}
		log.Print(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	funcMap := template.FuncMap{
		"toJSON": toJSON,
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(path)
	if err != nil {
		log.Printf("template parse error (%s): %v", path, err)
		http.Error(w, "template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, filepath.Base(path), data); err != nil {
		log.Printf("template execute error: %v", err)
		http.Error(w, "template execute error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Home gère la route racine `/`.
// Elle charge la liste des clubs depuis `data/clubs.json`, construit
// les données de page (`PageData`) et rend le template `index.html`.
// Si le chargement des clubs échoue, la liste est remplacée par une
// slice vide et l'erreur est loggée.
func Home(w http.ResponseWriter, r *http.Request) {
	clubs, err := models.LoadClubsFromFile("data/clubs.json")
	if err != nil {
		log.Printf("failed to load clubs: %v", err)

		clubs = []models.Club{}
	}
	data := PageData{
		Title:   "Accueil",
		Message: "Bienvenue sur la page d'accueil",
		Clubs:   clubs,
	}
	renderTemplate(w, "index.html", data)
}

// About gère la route `/about` et rend la page statique "À propos".
func About(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "À propos",
		Message: "Ceci est la page à propos",
	}
	renderTemplate(w, "about.html", data)
}

// Contact gère la route `/contact`.
// Pour une requête POST, elle lit les champs du formulaire (`name`, `msg`)
// et affiche un message de remerciement. Pour GET, elle affiche le
// formulaire de contact sans message.
func Contact(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		msg := r.FormValue("msg")

		data := PageData{
			Title:   "Contact",
			Message: "Merci " + name + " pour ton message : " + msg,
		}
		renderTemplate(w, "contact.html", data)
		return
	}

	data := PageData{
		Title:   "Contact",
		Message: "Envoie-nous un message",
	}
	renderTemplate(w, "contact.html", data)
}

// SearchAndFilter fournit l'endpoint `/api/clubs` en JSON.
// Elle charge tous les clubs, lit les paramètres de requête
// (`search`, `minYear`, `maxYear`, `page`, `pageSize`), applique
// les filtres de recherche et d'année, pagine les résultats,
// et renvoie un objet JSON contenant les clubs paginés et les métadonnées.
func SearchAndFilter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	clubs, err := models.LoadClubsFromFile("data/clubs.json")
	if err != nil {
		log.Printf("failed to load clubs: %v", err)
		clubs = []models.Club{}
	}

	
	search := strings.ToLower(r.URL.Query().Get("search"))
	minYear := r.URL.Query().Get("minYear")
	maxYear := r.URL.Query().Get("maxYear")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")

	
	page := 1
	pageSize := 6
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 50 {
		pageSize = ps
	}

	
	filtered := []models.Club{}
	for _, club := range clubs {
		// Search filter
		if search != "" {
			if !strings.Contains(strings.ToLower(club.Name), search) &&
				!strings.Contains(strings.ToLower(club.ShortName), search) &&
				!strings.Contains(strings.ToLower(club.TLA), search) {
				continue
			}
		}

		
		if minYear != "" {
			if min, err := strconv.Atoi(minYear); err == nil && club.Founded < min {
				continue
			}
		}
		if maxYear != "" {
			if max, err := strconv.Atoi(maxYear); err == nil && club.Founded > max {
				continue
			}
		}

		filtered = append(filtered, club)
	}

	total := len(filtered)
	totalPages := (total + pageSize - 1) / pageSize

	// Pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paged := filtered[start:end]

	response := FilterResponse{
		Clubs:      paged,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	json.NewEncoder(w).Encode(response)
}
