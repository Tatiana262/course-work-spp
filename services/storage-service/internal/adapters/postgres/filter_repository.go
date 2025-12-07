package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/core/domain"

	// "sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)


type FilterRepository struct {
	pool *pgxpool.Pool
}


func NewFilterRepository(pool *pgxpool.Pool) (*FilterRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("pgxpool.Pool cannot be nil")
	}
	return &FilterRepository{
		pool: pool,
	}, nil
}

// buildBaseQuery - хелпер для построения WHERE clause.
func (a *FilterRepository) buildFilterQuery(req domain.FilterOptions) (string, string, []interface{}) {
	joinClause := ""
	conditions := []string{"gp.is_source_duplicate = false", "gp.status = 'active'"}
	args := make([]interface{}, 0)
	argId := 1

	// Динамически добавляем JOIN, если он нужен для фильтрации или выборки
	switch req.Category {
	case "apartment":
		joinClause = "JOIN apartments d ON gp.id = d.property_id"
	case "house":
		joinClause = "JOIN houses d ON gp.id = d.property_id"
	case "room":
		joinClause = "JOIN rooms d ON gp.id = d.property_id"
	// ... и так далее для других категорий с таблицами деталей
	}

	// Добавляем фильтры по `general_properties`
	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("gp.category = $%d", argId))
		args = append(args, req.Category)
		argId++
	}
	if req.Region != "" {
		conditions = append(conditions, fmt.Sprintf("gp.region = $%d", argId))
		args = append(args, req.Region)
		argId++
	}
	if req.DealType != "" {
		conditions = append(conditions, fmt.Sprintf("gp.deal_type = $%d", argId))
		args = append(args, req.DealType)
		argId++
	}
	
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return joinClause, whereClause, args
}


// GetPriceRange получает минимальную и максимальную цену для заданных фильтров.
func (a *FilterRepository) GetPriceRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error) {
	_, whereClause, args := a.buildFilterQuery(req) // JOIN здесь не нужен
	
	query := fmt.Sprintf(`
		SELECT COALESCE(MIN(gp.price_byn), 0), COALESCE(MAX(gp.price_byn), 0) 
		FROM general_properties gp 
		%s`, whereClause,
	)

	var res domain.RangeResult
	err := a.pool.QueryRow(ctx, query, args...).Scan(&res.Min, &res.Max)
	if err != nil {
		return nil, fmt.Errorf("failed to get price range: %w", err)
	}
	return &res, nil
}

// GetDistinctRooms получает уникальные значения количества комнат.
func (a *FilterRepository) GetDistinctRooms(ctx context.Context, req domain.FilterOptions) ([]int, error) {
	joinClause, whereClause, args := a.buildFilterQuery(req)
	
	// Этот фильтр имеет смысл только для категорий, где есть `rooms_amount`
	if req.Category != "apartment" && req.Category != "house" && req.Category != "room" {
		return []int{}, nil
	}
	
	// `d` - это алиас для таблицы деталей (apartments, houses, etc.)
	query := fmt.Sprintf(`
		SELECT DISTINCT d.rooms_amount
		FROM general_properties gp
		%s
		%s AND d.rooms_amount IS NOT NULL AND d.rooms_amount > 0
		ORDER BY d.rooms_amount ASC
	`, joinClause, whereClause)
	
	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil { 
		return nil, fmt.Errorf("failed to get distinct rooms: %w", err)
	}
	defer rows.Close()

	var rooms []int
	for rows.Next() {
		var room int
		if err := rows.Scan(&room); err == nil {
			rooms = append(rooms, room)
		}
	}
	return rooms, rows.Err()
}


// GetDistinctWallMaterials получает уникальные значения материалов стен.
func (a *FilterRepository) GetDistinctWallMaterials(ctx context.Context, req domain.FilterOptions) ([]string, error) {
	joinClause, whereClause, args := a.buildFilterQuery(req)
	
	if req.Category != "apartment" && req.Category != "house" {
		return []string{}, nil
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT d.wall_material
		FROM general_properties gp
		%s
		%s AND d.wall_material IS NOT NULL AND d.wall_material != ''
		ORDER BY d.wall_material ASC
	`, joinClause, whereClause)

	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil { 
		return nil, fmt.Errorf("failed to get distinct wall materials: %w", err)
	}
	defer rows.Close()

	var materials []string
	for rows.Next() {
		var material string
		if err := rows.Scan(&material); err == nil {
			materials = append(materials, material)
		}
	}
	return materials, rows.Err()
}


// GetYearBuiltRange получает диапазон годов постройки.
func (a *FilterRepository) GetYearBuiltRange(ctx context.Context, req domain.FilterOptions) (*domain.RangeResult, error) {
	joinClause, whereClause, args := a.buildFilterQuery(req)

	if req.Category != "apartment" && req.Category != "house" {
		return &domain.RangeResult{}, nil
	}
	
	query := fmt.Sprintf(`
		SELECT COALESCE(MIN(d.year_built), 0), COALESCE(MAX(d.year_built), 0) 
		FROM general_properties gp
		%s
		%s AND d.year_built IS NOT NULL AND d.year_built > 1800 -- Отсекаем мусорные значения
	`, joinClause, whereClause)

	var res domain.RangeResult
	err := a.pool.QueryRow(ctx, query, args...).Scan(&res.Min, &res.Max)
	if err != nil {
		return nil, fmt.Errorf("failed to get year built range: %w", err)
	}
	return &res, nil
}



// GetUniqueCategories извлекает уникальные категории и их русские названия.
func (a *FilterRepository) GetUniqueCategories(ctx context.Context) ([]domain.DictionaryItem, error) {
	query := `
		SELECT DISTINCT category 
		FROM general_properties 
		WHERE status = 'active' AND is_source_duplicate = false AND category IS NOT NULL AND category != ''
		ORDER BY category
	`
	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique categories: %w", err)
	}
	defer rows.Close()

	var items []domain.DictionaryItem
	for rows.Next() {
		var systemName string
		if err := rows.Scan(&systemName); err == nil {
			items = append(items, domain.DictionaryItem{
				SystemName:  systemName,
				DisplayName: translateCategory(systemName), // Используем хелпер для перевода
			})
		}
	}
	return items, rows.Err()
}


// GetUniqueRegions извлекает уникальные регионы. Для них системное и отображаемое имя совпадают.
func (a *FilterRepository) GetUniqueRegions(ctx context.Context) ([]domain.DictionaryItem, error) {
	query := `
		SELECT DISTINCT region
		FROM general_properties
		WHERE status = 'active' AND is_source_duplicate = false AND region IS NOT NULL AND region != ''
		ORDER BY region
	`
	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique regions: %w", err)
	}
	defer rows.Close()

	var items []domain.DictionaryItem
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			items = append(items, domain.DictionaryItem{
				SystemName:  name,
				DisplayName: name, // Для регионов они совпадают
			})
		}
	}
	return items, rows.Err()
}

// GetUniqueDealTypes извлекает уникальные типы сделок.
func (a *FilterRepository) GetUniqueDealTypes(ctx context.Context) ([]domain.DictionaryItem, error) {
	query := `
		SELECT DISTINCT deal_type
		FROM general_properties
		WHERE status = 'active' AND is_source_duplicate = false AND deal_type IS NOT NULL
		ORDER BY deal_type
	`
	rows, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique deal types: %w", err)
	}
	defer rows.Close()

	var items []domain.DictionaryItem
	for rows.Next() {
		var systemName string
		if err := rows.Scan(&systemName); err == nil {
			items = append(items, domain.DictionaryItem{
				SystemName:  systemName,
				DisplayName: translateDealType(systemName), // Хелпер для перевода
			})
		}
	}
	return items, rows.Err()
}



func translateCategory(systemName string) string {
	// Этот "словарь" можно вынести в пакет dictionaries, как мы обсуждали
	translations := map[string]string{
		"apartment":    "Квартиры",
		"house":        "Дома",
		"commercial":   "Коммерция",
		"plot":         "Участки",
		"room":         "Комнаты",
		"new_building": "Новостройки",
		"garage":       "Гаражи",
	}
	if val, ok := translations[systemName]; ok {
		return val
	}
	return systemName // Fallback
}

func translateDealType(systemName string) string {
	switch systemName {
	case "sale":
		return "Продажа"
	case "rent":
		return "Аренда"
	default:
		return systemName
	}
}