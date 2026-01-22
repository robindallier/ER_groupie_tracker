package controller

import (
	"encoding/json"
	"fmt"
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
	Title       string
	Message     string
	Clubs       []models.Club
	Favorites   []models.Club
	FavoriteIDs map[string]bool
	SearchQuery string
	MinYear     string
	MaxYear     string
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

// GetFavoritesFromCookie lit les favoris depuis le cookie "favorites" de l'utilisateur.
// Le cookie contient une liste d'IDs de clubs séparés par des virgules (ex: "12,34,56").
// Cette fonction gère les cas où le cookie n'existe pas ou est vide en renvoyant
// une slice vide. Elle retourne toujours une slice de chaînes d'identifiants.
func GetFavoritesFromCookie(r *http.Request) []string {
	cookie, err := r.Cookie("favorites")
	if err != nil {
		return []string{}
	}
	if cookie.Value == "" {
		return []string{}
	}
	return strings.Split(cookie.Value, ",")
}

// AddFavorite ajoute un club aux favoris.
// Attendu: requête HTTP POST avec le champ de formulaire `club_id`.
// Comportement:
//   - Valide que la méthode est POST et que `club_id` est fourni.
//   - Lit le cookie `favorites` existant (liste d'IDs séparés par des virgules).
//   - Si l'ID n'est pas déjà présent, l'ajoute à la liste et remet à jour le cookie
//     avec une durée de vie de 30 jours.
//   - Redirige ensuite vers la page précédente (en utilisant l'en-tête Referer)
//     ou vers l'URL par défaut fournie.
func AddFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		redirectBack(w, r, "/")
		return
	}

	clubID := r.FormValue("club_id")
	if clubID == "" {
		redirectBack(w, r, "/")
		return
	}

	favorites := GetFavoritesFromCookie(r)

	// Vérifier si le club n'est pas déjà dans les favoris
	for _, fav := range favorites {
		if fav == clubID {
			redirectBack(w, r, "/")
			return
		}
	}

	favorites = append(favorites, clubID)

	cookie := &http.Cookie{
		Name:     "favorites",
		Value:    strings.Join(favorites, ","),
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 jours
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	redirectBack(w, r, "/")
}

// RemoveFavorite supprime un club des favoris.
// Attendu: requête HTTP POST avec le champ de formulaire `club_id`.
// Comportement:
//   - Valide que la méthode est POST et que `club_id` est fourni.
//   - Lit le cookie `favorites`, retire l'ID fourni s'il y est présent,
//     puis réécrit le cookie avec la nouvelle liste.
//   - La durée du cookie reste identique (30 jours) ; si la liste devient vide,
//     le cookie est mis à jour en conséquence.
//   - Redirige ensuite vers la page précédente (Referer) ou vers l'URL par défaut.
func RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		redirectBack(w, r, "/")
		return
	}

	clubID := r.FormValue("club_id")
	if clubID == "" {
		redirectBack(w, r, "/")
		return
	}

	favorites := GetFavoritesFromCookie(r)
	newFavorites := []string{}

	for _, fav := range favorites {
		if fav != clubID {
			newFavorites = append(newFavorites, fav)
		}
	}

	cookie := &http.Cookie{
		Name:     "favorites",
		Value:    strings.Join(newFavorites, ","),
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	redirectBack(w, r, "/")
}

// redirectBack redirige vers la page précédente en utilisant l'en-tête HTTP
// `Referer`. Si l'en-tête n'est pas présent, la fonction redirige vers
// `defaultURL`. Utilise le statut HTTP 303 (See Other) pour les redirections
// après un POST afin d'éviter la re-soumission de formulaire par le navigateur.
func redirectBack(w http.ResponseWriter, r *http.Request, defaultURL string) {
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = defaultURL
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

// HomeWithFavorites affiche la page d'accueil en tenant compte des favoris
// et des paramètres de recherche/filtres passés par la requête GET.
// Étapes réalisées:
//  1. Charge tous les clubs depuis `data/clubs.json`.
//  2. Récupère les paramètres GET `search`, `minYear`, `maxYear` et applique
//     les filtres côté serveur (recherche textuelle et plage d'années).
//  3. Lit le cookie `favorites` et construit une map `FavoriteIDs` pour
//     indiquer rapidement si un club est favori (utile dans le template).
//  4. Prépare le `PageData` avec : les clubs filtrés, la liste des favoris,
//     la map des IDs favoris et les valeurs de recherche pour pré-remplir le formulaire.
//  5. Rend le template `index.html`.
func HomeWithFavorites(w http.ResponseWriter, r *http.Request) {
	clubs, err := models.LoadClubsFromFile("data/clubs.json")
	if err != nil {
		log.Printf("failed to load clubs: %v", err)
		clubs = []models.Club{}
	}

	// Récupérer les paramètres de recherche
	search := strings.ToLower(r.URL.Query().Get("search"))
	minYearStr := r.URL.Query().Get("minYear")
	maxYearStr := r.URL.Query().Get("maxYear")

	// Filtrer les clubs
	filteredClubs := []models.Club{}
	for _, club := range clubs {
		// Search filter
		if search != "" {
			if !strings.Contains(strings.ToLower(club.Name), search) &&
				!strings.Contains(strings.ToLower(club.ShortName), search) &&
				!strings.Contains(strings.ToLower(club.TLA), search) {
				continue
			}
		}

		// Min year filter
		if minYearStr != "" {
			if minYear, err := strconv.Atoi(minYearStr); err == nil && club.Founded < minYear {
				continue
			}
		}

		// Max year filter
		if maxYearStr != "" {
			if maxYear, err := strconv.Atoi(maxYearStr); err == nil && club.Founded > maxYear {
				continue
			}
		}

		filteredClubs = append(filteredClubs, club)
	}

	// Récupérer les IDs des favoris
	favoriteIDs := GetFavoritesFromCookie(r)
	favoriteIDMap := make(map[string]bool)
	for _, id := range favoriteIDs {
		favoriteIDMap[id] = true
	}

	// Construire la liste des clubs favoris
	favorites := []models.Club{}
	for _, club := range clubs {
		if favoriteIDMap[fmt.Sprintf("%d", club.ID)] {
			favorites = append(favorites, club)
		}
	}

	data := PageData{
		Title:       "Accueil",
		Message:     "Bienvenue sur la page d'accueil",
		Clubs:       filteredClubs,
		Favorites:   favorites,
		FavoriteIDs: favoriteIDMap,
		SearchQuery: search,
		MinYear:     minYearStr,
		MaxYear:     maxYearStr,
	}
	renderTemplate(w, "index.html", data)
}

// Favorites affiche la page listant uniquement les clubs marqués comme favoris.
// Fonctionnement:
//   - Charge tous les clubs depuis `data/clubs.json`.
//   - Lit le cookie `favorites` et construit une map d'IDs favorisés.
//   - Construit la slice `favorites` contenant les objets `models.Club`
//     correspondant aux IDs favoris.
//   - Rend le template `favorites.html` avec `PageData.Favorites`.
func Favorites(w http.ResponseWriter, r *http.Request) {
	clubs, err := models.LoadClubsFromFile("data/clubs.json")
	if err != nil {
		log.Printf("failed to load clubs: %v", err)
		clubs = []models.Club{}
	}

	// Récupérer les IDs des favoris
	favoriteIDs := GetFavoritesFromCookie(r)
	favoriteIDMap := make(map[string]bool)
	for _, id := range favoriteIDs {
		favoriteIDMap[id] = true
	}

	// Construire la liste des clubs favoris
	favorites := []models.Club{}
	for _, club := range clubs {
		if favoriteIDMap[fmt.Sprintf("%d", club.ID)] {
			favorites = append(favorites, club)
		}
	}

	data := PageData{
		Title:       "Mes Favoris",
		Message:     "Vos clubs favoris",
		Favorites:   favorites,
		FavoriteIDs: favoriteIDMap,
	}
	renderTemplate(w, "favorites.html", data)
}

// ClearFavorites supprime tous les favoris enregistrés pour l'utilisateur.
// Attendu: requête POST. La fonction réinitialise le cookie `favorites`
// en le vidant (MaxAge=-1) pour effacer la valeur côté client, puis
// redirige vers la page `/favorites`.
func ClearFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/favorites", http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:     "favorites",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/favorites", http.StatusSeeOther)
}
