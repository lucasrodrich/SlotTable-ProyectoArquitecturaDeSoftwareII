package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blassardoy/restaurant-reservas/search-api/internal/cache"
	"github.com/blassardoy/restaurant-reservas/search-api/internal/domain"
	"github.com/blassardoy/restaurant-reservas/search-api/internal/repository"
)

// Mock implementations

type mockSearchRepository struct {
	searchFunc        func(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error)
	getByIDFunc       func(ctx context.Context, id string) (*domain.TableAvailability, error)
	indexFunc         func(ctx context.Context, doc domain.TableAvailability) error
	updateFunc        func(ctx context.Context, doc domain.TableAvailability) error
	deleteFunc        func(ctx context.Context, id string) error
	deleteByQueryFunc func(ctx context.Context, query string) error
}

func (m *mockSearchRepository) Search(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, q)
	}
	return &repository.SearchResult{
		Results: []domain.TableAvailability{},
		Total:   0,
		Page:    1,
		Size:    10,
		Pages:   0,
	}, nil
}

func (m *mockSearchRepository) GetByID(ctx context.Context, id string) (*domain.TableAvailability, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockSearchRepository) Index(ctx context.Context, doc domain.TableAvailability) error {
	if m.indexFunc != nil {
		return m.indexFunc(ctx, doc)
	}
	return nil
}

func (m *mockSearchRepository) Update(ctx context.Context, doc domain.TableAvailability) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, doc)
	}
	return nil
}

func (m *mockSearchRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockSearchRepository) DeleteByQuery(ctx context.Context, query string) error {
	if m.deleteByQueryFunc != nil {
		return m.deleteByQueryFunc(ctx, query)
	}
	return nil
}

// Helper functions to create mock clients

func createMockReservationClient() *ReservationClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	return NewReservationClient(server.URL)
}

func createMockTableClient() *TableClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"1","table_number":1,"capacity":4,"meal_type":"lunch"}]`))
	}))
	return NewTableClient(server.URL)
}

// Tests

func TestSearchService_Search_Success(t *testing.T) {
	expectedResults := []domain.TableAvailability{
		{
			ID:          "table-lunch-1-2024-01-15",
			TableNumber: 1,
			Capacity:    4,
			MealType:    "lunch",
			Date:        "2024-01-15",
			IsAvailable: true,
		},
	}

	mockRepo := &mockSearchRepository{
		searchFunc: func(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error) {
			return &repository.SearchResult{
				Results: expectedResults,
				Total:   1,
				Page:    1,
				Size:    10,
				Pages:   1,
			}, nil
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	query := repository.SearchQuery{
		Q:    "lunch",
		Page: 1,
		Size: 10,
	}

	result, err := service.Search(context.Background(), query)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}

	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestSearchService_Search_WithCache(t *testing.T) {
	callCount := 0

	mockRepo := &mockSearchRepository{
		searchFunc: func(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error) {
			callCount++
			return &repository.SearchResult{
				Results: []domain.TableAvailability{},
				Total:   0,
				Page:    1,
				Size:    10,
				Pages:   0,
			}, nil
		},
	}

	// Non-zero TTL: this test asserts a cached value stays retrievable, so a
	// zero TTL (expires = time.Now() at Set time) would make it always miss -
	// deterministically, not just flaky, once the clock has real resolution.
	cacheLayer := cache.NewDual(time.Minute, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	query := repository.SearchQuery{
		Q:    "test",
		Page: 1,
		Size: 10,
	}

	// First call - should hit repository
	_, err := service.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Second call - should hit cache
	_, err = service.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should only call repository once due to cache
	if callCount != 1 {
		t.Errorf("expected 1 repository call, got %d", callCount)
	}
}

func TestSearchService_GetByID_Success(t *testing.T) {
	expectedTable := &domain.TableAvailability{
		ID:          "table-lunch-1-2024-01-15",
		TableNumber: 1,
		Capacity:    4,
		MealType:    "lunch",
		Date:        "2024-01-15",
		IsAvailable: true,
	}

	mockRepo := &mockSearchRepository{
		getByIDFunc: func(ctx context.Context, id string) (*domain.TableAvailability, error) {
			if id == "table-lunch-1-2024-01-15" {
				return expectedTable, nil
			}
			return nil, errors.New("not found")
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	result, err := service.GetByID(context.Background(), "table-lunch-1-2024-01-15")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.ID != "table-lunch-1-2024-01-15" {
		t.Errorf("expected ID 'table-lunch-1-2024-01-15', got %s", result.ID)
	}
}

func TestSearchService_GetByID_NotFound(t *testing.T) {
	mockRepo := &mockSearchRepository{
		getByIDFunc: func(ctx context.Context, id string) (*domain.TableAvailability, error) {
			return nil, errors.New("not found")
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	_, err := service.GetByID(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearchService_Stats_Success(t *testing.T) {
	mockRepo := &mockSearchRepository{
		searchFunc: func(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error) {
			return &repository.SearchResult{
				Total: 42,
				Page:  1,
				Size:  1,
				Pages: 42,
			}, nil
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	stats, err := service.Stats(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats.Documents != 42 {
		t.Errorf("expected 42 documents, got %d", stats.Documents)
	}
}

func TestSearchService_InvalidateAll(t *testing.T) {
	mockRepo := &mockSearchRepository{}
	cacheLayer := cache.NewDual(time.Minute, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	// Set a value in cache first
	key := "test-key"
	cacheLayer.Set(key, "test-value")

	// Verify it's in cache
	if _, ok := cacheLayer.Get(key); !ok {
		t.Fatal("expected value in cache before invalidation")
	}

	// Invalidate all
	service.InvalidateAll()

	// Verify it's removed
	if _, ok := cacheLayer.Get(key); ok {
		t.Error("expected cache to be cleared")
	}
}

func TestSearchService_GetCacheValue(t *testing.T) {
	mockRepo := &mockSearchRepository{}
	cacheLayer := cache.NewDual(time.Minute, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	key := "test-key"
	value := "test-value"
	cacheLayer.Set(key, value)

	result, ok := service.GetCacheValue(key)

	if !ok {
		t.Fatal("expected to find value in cache")
	}

	if result != value {
		t.Errorf("expected value '%s', got '%v'", value, result)
	}
}

func TestSearchService_GetCacheValue_NotFound(t *testing.T) {
	mockRepo := &mockSearchRepository{}
	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	_, ok := service.GetCacheValue("nonexistent")

	if ok {
		t.Error("expected not to find value in cache")
	}
}

func TestSearchService_Reindex_Success(t *testing.T) {
	deleteByQueryCalled := false
	indexedTables := []domain.TableAvailability{}

	mockRepo := &mockSearchRepository{
		deleteByQueryFunc: func(ctx context.Context, query string) error {
			deleteByQueryCalled = true
			return nil
		},
		indexFunc: func(ctx context.Context, doc domain.TableAvailability) error {
			indexedTables = append(indexedTables, doc)
			return nil
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	resClient := createMockReservationClient()
	tableClient := createMockTableClient()
	service := NewSearchService(mockRepo, cacheLayer, resClient, tableClient, 7)

	err := service.Reindex(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !deleteByQueryCalled {
		t.Error("expected DeleteByQuery to be called")
	}

	if len(indexedTables) == 0 {
		t.Error("expected tables to be indexed")
	}
}

func TestSearchService_Reindex_NoTableClient(t *testing.T) {
	mockRepo := &mockSearchRepository{
		deleteByQueryFunc: func(ctx context.Context, query string) error {
			return nil
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	err := service.Reindex(context.Background())

	if err == nil {
		t.Fatal("expected error when table client is nil, got nil")
	}

	if err.Error() != "table client not configured" {
		t.Errorf("expected 'table client not configured' error, got %v", err)
	}
}

func TestSearchService_WindowDays_Default(t *testing.T) {
	mockRepo := &mockSearchRepository{}
	cacheLayer := cache.NewDual(0, nil, nil)
	service := &searchService{
		repo:             mockRepo,
		cache:            cacheLayer,
		availabilityDays: 0, // Should default to 30
	}

	days := service.windowDays()

	if days != 30 {
		t.Errorf("expected default 30 days, got %d", days)
	}
}

func TestSearchService_WindowDays_Custom(t *testing.T) {
	mockRepo := &mockSearchRepository{}
	cacheLayer := cache.NewDual(0, nil, nil)
	service := &searchService{
		repo:             mockRepo,
		cache:            cacheLayer,
		availabilityDays: 60,
	}

	days := service.windowDays()

	if days != 60 {
		t.Errorf("expected 60 days, got %d", days)
	}
}

func TestSearchService_Search_RepositoryError(t *testing.T) {
	mockRepo := &mockSearchRepository{
		searchFunc: func(ctx context.Context, q repository.SearchQuery) (*repository.SearchResult, error) {
			return nil, errors.New("repository error")
		},
	}

	cacheLayer := cache.NewDual(0, nil, nil)
	service := NewSearchService(mockRepo, cacheLayer, nil, nil, 30)

	query := repository.SearchQuery{
		Q:    "test",
		Page: 1,
		Size: 10,
	}

	_, err := service.Search(context.Background(), query)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "repository error" {
		t.Errorf("expected 'repository error', got %v", err)
	}
}

func TestSearchService_Reindex_WithReservations(t *testing.T) {
	indexedTables := []domain.TableAvailability{}

	mockRepo := &mockSearchRepository{
		deleteByQueryFunc: func(ctx context.Context, query string) error {
			return nil
		},
		indexFunc: func(ctx context.Context, doc domain.TableAvailability) error {
			indexedTables = append(indexedTables, doc)
			return nil
		},
	}

	// Mock server that returns a reservation
	reservationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		futureDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		response := `[{"id":"res1","owner_id":"1","table_number":1,"guests":4,"date_time":"` + futureDate + `","meal_type":"lunch","status":"confirmed","total_price":100.0}]`
		w.Write([]byte(response))
	}))
	defer reservationServer.Close()

	tableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"1","table_number":1,"capacity":4,"meal_type":"lunch"}]`))
	}))
	defer tableServer.Close()

	cacheLayer := cache.NewDual(0, nil, nil)
	resClient := NewReservationClient(reservationServer.URL)
	tableClient := NewTableClient(tableServer.URL)
	service := NewSearchService(mockRepo, cacheLayer, resClient, tableClient, 7)

	err := service.Reindex(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have indexed tables (7 days)
	if len(indexedTables) == 0 {
		t.Error("expected tables to be indexed")
	}

	// Check if reservation affected availability
	hasUnavailableTable := false
	for _, table := range indexedTables {
		if !table.IsAvailable {
			hasUnavailableTable = true
			break
		}
	}

	if !hasUnavailableTable {
		t.Error("expected at least one unavailable table due to reservation")
	}
}
