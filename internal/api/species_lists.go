package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// List types for species management
const (
	ListTypeConfirmed   = "confirmed"
	ListTypeExcluded    = "excluded"
	ListTypeWhitelisted = "whitelisted"
	ListTypeInclude     = "include"
)

// SpeciesListsResponse represents all species lists.
type SpeciesListsResponse struct {
	Confirmed   []string `json:"confirmed"`
	Excluded    []string `json:"excluded"`
	Whitelisted []string `json:"whitelisted"`
	Include     []string `json:"include"`
}

// SpeciesListRequest is used for add/remove operations.
type SpeciesListRequest struct {
	Species string `json:"species"`
}

// UpdateSpeciesListRequest is used for full list replacement.
type UpdateSpeciesListRequest struct {
	Species []string `json:"species"`
}

// LabelsResponse represents all available species labels.
type LabelsResponse struct {
	Labels []string `json:"labels"`
	Total  int      `json:"total"`
}

// getListFilePath returns the full path to a species list file.
func (h *Handlers) getListFilePath(listType string) string {
	switch listType {
	case ListTypeConfirmed:
		return filepath.Join(h.scriptsDir, "confirmed_species_list.txt")
	case ListTypeExcluded:
		return filepath.Join(h.scriptsDir, "exclude_species_list.txt")
	case ListTypeWhitelisted:
		return filepath.Join(h.scriptsDir, "whitelist_species_list.txt")
	case ListTypeInclude:
		return filepath.Join(h.scriptsDir, "include_species_list.txt")
	default:
		return ""
	}
}

// readSpeciesListFile reads a species list file and returns its contents.
func readSpeciesListFile(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var species []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			species = append(species, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return species, nil
}

// writeSpeciesListFile writes a species list to a file.
func writeSpeciesListFile(path string, species []string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, s := range species {
		if _, err := file.WriteString(s + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// GetSpeciesLists handles GET /api/species-lists requests.
// Returns all species lists (confirmed, excluded, whitelisted, include).
func (h *Handlers) GetSpeciesLists(w http.ResponseWriter, r *http.Request) {
	confirmed, err := readSpeciesListFile(h.getListFilePath(ListTypeConfirmed))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read confirmed species list")
		return
	}

	excluded, err := readSpeciesListFile(h.getListFilePath(ListTypeExcluded))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read excluded species list")
		return
	}

	whitelisted, err := readSpeciesListFile(h.getListFilePath(ListTypeWhitelisted))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read whitelisted species list")
		return
	}

	include, err := readSpeciesListFile(h.getListFilePath(ListTypeInclude))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read include species list")
		return
	}

	// Ensure empty arrays instead of null
	if confirmed == nil {
		confirmed = []string{}
	}
	if excluded == nil {
		excluded = []string{}
	}
	if whitelisted == nil {
		whitelisted = []string{}
	}
	if include == nil {
		include = []string{}
	}

	response := SpeciesListsResponse{
		Confirmed:   confirmed,
		Excluded:    excluded,
		Whitelisted: whitelisted,
		Include:     include,
	}

	writeJSON(w, http.StatusOK, response)
}

// validateListType checks if the list type is valid.
func validateListType(listType string) bool {
	switch listType {
	case ListTypeConfirmed, ListTypeExcluded, ListTypeWhitelisted, ListTypeInclude:
		return true
	default:
		return false
	}
}

// AddToSpeciesList handles POST /api/species-lists/{listType}/add requests.
// Adds a species to the specified list.
func (h *Handlers) AddToSpeciesList(w http.ResponseWriter, r *http.Request) {
	listType := chi.URLParam(r, "listType")
	if !validateListType(listType) {
		writeError(w, http.StatusBadRequest, "Invalid list type")
		return
	}

	var req SpeciesListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Species == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	filePath := h.getListFilePath(listType)
	species, err := readSpeciesListFile(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read species list")
		return
	}

	// Check if species already exists
	for _, s := range species {
		if s == req.Species {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Species already in list"})
			return
		}
	}

	// Add species to list
	species = append(species, req.Species)

	if err := writeSpeciesListFile(filePath, species); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write species list")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RemoveFromSpeciesList handles POST /api/species-lists/{listType}/remove requests.
// Removes a species from the specified list.
func (h *Handlers) RemoveFromSpeciesList(w http.ResponseWriter, r *http.Request) {
	listType := chi.URLParam(r, "listType")
	if !validateListType(listType) {
		writeError(w, http.StatusBadRequest, "Invalid list type")
		return
	}

	var req SpeciesListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Species == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	filePath := h.getListFilePath(listType)
	species, err := readSpeciesListFile(filePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read species list")
		return
	}

	// Remove species from list
	var newSpecies []string
	found := false
	for _, s := range species {
		if s != req.Species {
			newSpecies = append(newSpecies, s)
		} else {
			found = true
		}
	}

	if !found {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Species not in list"})
		return
	}

	if err := writeSpeciesListFile(filePath, newSpecies); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write species list")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdateSpeciesList handles PUT /api/species-lists/{listType} requests.
// Replaces the entire species list.
func (h *Handlers) UpdateSpeciesList(w http.ResponseWriter, r *http.Request) {
	listType := chi.URLParam(r, "listType")
	if !validateListType(listType) {
		writeError(w, http.StatusBadRequest, "Invalid list type")
		return
	}

	var req UpdateSpeciesListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	filePath := h.getListFilePath(listType)
	if err := writeSpeciesListFile(filePath, req.Species); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write species list")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetLabels handles GET /api/labels requests.
// Returns all available species labels from labels.txt.
func (h *Handlers) GetLabels(w http.ResponseWriter, r *http.Request) {
	labelsPath := filepath.Join(h.scriptsDir, "labels.txt")

	labels, err := readSpeciesListFile(labelsPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read labels file")
		return
	}

	if labels == nil {
		labels = []string{}
	}

	response := LabelsResponse{
		Labels: labels,
		Total:  len(labels),
	}

	writeJSON(w, http.StatusOK, response)
}
