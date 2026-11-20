package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

//go:embed help/conformance-openapi.yaml
var conformanceOpenAPIYAML []byte

//go:embed help/conformance-environment.yaml
var conformanceEnvironmentYAML []byte

// The Conformance Server is deliberately imperfect: every operation in
// help/conformance-openapi.yaml conforms except getProduct, listCategories and
// listOrders, which each break the spec in a different way. conformanceKnownViolations
// is the single list that the tests and the user guide rely on.
var conformanceKnownViolations = []string{"getProduct", "listCategories", "listOrders"}

type conformanceProduct struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type conformanceServer struct {
	mu       sync.Mutex
	products []conformanceProduct
	nextID   int
}

func newConformanceMux() http.Handler {
	s := &conformanceServer{
		products: []conformanceProduct{{1, "Keyboard", 49.9}, {2, "Mouse", 19.5}},
		nextID:   3,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Write(conformanceOpenAPIYAML) //nolint:errcheck
	})
	mux.HandleFunc("/environment.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Write(conformanceEnvironmentYAML) //nolint:errcheck
	})
	mux.HandleFunc("/docs", handleDemoSwaggerUI)
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/products", s.handleProducts)
	mux.HandleFunc("/products/", s.product)
	mux.HandleFunc("/categories", s.categories)
	mux.HandleFunc("/orders", s.orders)
	return corsMiddleware(mux)
}

func (s *conformanceServer) health(w http.ResponseWriter, r *http.Request) {
	demoWriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *conformanceServer) handleProducts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Method {
	case http.MethodGet:
		demoWriteJSON(w, http.StatusOK, s.products)
	case http.MethodPost:
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
			demoErrJSON(w, http.StatusBadRequest, "name is required")
			return
		}
		p := conformanceProduct{ID: s.nextID, Name: in.Name, Price: 9.99}
		s.nextID++
		s.products = append(s.products, p)
		demoWriteJSON(w, http.StatusCreated, p)
	default:
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *conformanceServer) product(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, p := range s.products {
		if "/products/"+strconv.Itoa(p.ID) == strings.TrimSuffix(r.URL.Path, "/") {
			idx = i
		}
	}
	if idx < 0 {
		demoErrJSON(w, http.StatusNotFound, "product not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		p := s.products[idx]
		demoWriteJSON(w, http.StatusOK, map[string]any{"id": p.ID, "name": p.Name, "price": strconv.FormatFloat(p.Price, 'f', 2, 64)})
	case http.MethodDelete:
		s.products = append(s.products[:idx], s.products[idx+1:]...)
		w.WriteHeader(http.StatusNoContent)
	default:
		demoErrJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *conformanceServer) categories(w http.ResponseWriter, r *http.Request) {
	demoWriteJSON(w, http.StatusOK, []map[string]any{{"id": 1}, {"id": 2}})
}

func (s *conformanceServer) orders(w http.ResponseWriter, r *http.Request) {
	demoWriteJSON(w, http.StatusOK, []map[string]any{{"id": "1", "total": 19.5}})
}
