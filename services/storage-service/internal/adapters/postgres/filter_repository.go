package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/core/domain"

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


// getDistinctValues - для получения уникальных значений (строк или чисел).
func (r *FilterRepository) getDistinctValues(ctx context.Context, tableName, columnName string) ([]interface{}, error) {
	// Для получения опций мы НЕ используем фильтры по региону и т.д.
	// Только по is_source_duplicate и status.
	// JOIN нужен, чтобы связать с `general_properties` и проверить статус.
	// query := fmt.Sprintf(`
	// 	SELECT DISTINCT d.%s
	// 	FROM %s d
	// 	JOIN general_properties gp ON d.property_id = gp.id
	// 	WHERE gp.is_source_duplicate = false
	// 	  AND gp.status = 'active'
	// 	  AND d.%s IS NOT NULL
	// 	ORDER BY d.%s ASC
	// `, columnName, tableName, columnName, columnName)

	query := fmt.Sprintf(`
		WITH latest_visible_objects AS (
			SELECT
				gp.id,
				ROW_NUMBER() OVER(PARTITION BY gp.master_object_id ORDER BY gp.updated_at DESC) as rn
			FROM
				general_properties gp
			WHERE
				gp.is_source_duplicate = false AND gp.status = 'active'
		)
		SELECT DISTINCT d.%s
		FROM %s d
		JOIN latest_visible_objects lvo ON d.property_id = lvo.id
		WHERE
			lvo.rn = 1
			AND d.%s IS NOT NULL AND d.%s::text != ''
		ORDER BY d.%s ASC
	`, columnName, tableName, columnName, columnName, columnName)
	
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct values for %s.%s: %w", tableName, columnName, err)
	}
	defer rows.Close()

	var values []interface{}
	for rows.Next() {
		var val interface{}
		if err := rows.Scan(&val); err == nil {
			values = append(values, val)
		}
	}
	return values, rows.Err()
}

// getRange - для получения MIN/MAX.
func (r *FilterRepository) getRange(ctx context.Context, tableName, columnName string) (*domain.RangeResult, error) {
	// query := fmt.Sprintf(`
	// 	SELECT COALESCE(MIN(d.%s), 0), COALESCE(MAX(d.%s), 0)
	// 	FROM %s d
	// 	JOIN general_properties gp ON d.property_id = gp.id
	// 	WHERE gp.is_source_duplicate = false
	// 	  AND gp.status = 'active'
	// 	  AND d.%s IS NOT NULL
	// `, columnName, columnName, tableName, columnName)
	
	query := fmt.Sprintf(`
		WITH latest_visible_objects AS (
			SELECT
				gp.id,
				ROW_NUMBER() OVER(PARTITION BY gp.master_object_id ORDER BY gp.updated_at DESC) as rn
			FROM
				general_properties gp
			WHERE
				gp.is_source_duplicate = false AND gp.status = 'active'
		)
		SELECT
			COALESCE(MIN(d.%s), 0),
			COALESCE(MAX(d.%s), 0)
		FROM
			%s d
		JOIN latest_visible_objects lvo ON d.property_id = lvo.id
		WHERE
			lvo.rn = 1
			AND d.%s IS NOT NULL
	`, columnName, columnName, tableName, columnName)

	var res domain.RangeResult
	err := r.pool.QueryRow(ctx, query).Scan(&res.Min, &res.Max)
	if err != nil {
		return nil, fmt.Errorf("failed to get range for %s.%s: %w", tableName, columnName, err)
	}
	return &res, nil
}

func (a *FilterRepository) GetPriceRange(ctx context.Context, req domain.FindObjectsFilters) (*domain.RangeResult, error) {
	// 1. Получаем базовые WHERE и JOIN от нашего билдера.
    // Нам пока не нужен JOIN для цены, но получим его на случай будущих доработок.
	joinClause, whereClause, args := applyFilters(req)

	var priceColumn string
	switch req.PriceCurrency {
	case "USD":
		priceColumn = "gp.price_usd"
	case "EUR":
		priceColumn = "gp.price_eur"
	case "BYN":
		priceColumn = "gp.price_byn"
	default:
		// Валюта по умолчанию, если не указана.
		// Лучше всего использовать ту, которая является основной для вашего сайта.
		priceColumn = "gp.price_usd" 
	}

	// 2. Формируем "умный" SQL-запрос.
	query := fmt.Sprintf(`
		WITH latest_visible_objects AS (
			SELECT
				gp.id,
				%s AS price, -- Выбираем только те поля, которые нужны
				ROW_NUMBER() OVER(PARTITION BY gp.master_object_id ORDER BY gp.updated_at DESC) as rn
			FROM
				general_properties gp
				%s
			%s -- Вставляем WHERE с фильтрами (region, category, etc.)
		)
		SELECT
			COALESCE(MIN(price), 0),
			COALESCE(MAX(price), 0)
		FROM
			latest_visible_objects
		WHERE
			rn = 1;
	`, priceColumn, joinClause, whereClause)

	var res domain.RangeResult
	err := a.pool.QueryRow(ctx, query, args...).Scan(&res.Min, &res.Max)
	if err != nil {
		return nil, fmt.Errorf("failed to get smart price range: %w", err)
	}
	return &res, nil
}

func (a *FilterRepository) getDictionary(ctx context.Context, field string) ([]domain.DictionaryItem, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT %s 
		FROM general_properties 
		WHERE status = 'active' AND is_source_duplicate = false
		ORDER BY %s
	`, field, field)
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
				DisplayName: translator(field, systemName), // Используем хелпер для перевода
			})
		}
	}
	return items, rows.Err()
}


// GetUniqueCategories извлекает уникальные категории и их русские названия.
func (a *FilterRepository) GetUniqueCategories(ctx context.Context) ([]domain.DictionaryItem, error) {
	return a.getDictionary(ctx, "category")
}


// GetUniqueRegions извлекает уникальные регионы. Для них системное и отображаемое имя совпадают.
func (a *FilterRepository) GetUniqueRegions(ctx context.Context) ([]domain.DictionaryItem, error) {
	return a.getDictionary(ctx, "region")
}

// GetUniqueDealTypes извлекает уникальные типы сделок.
func (a *FilterRepository) GetUniqueDealTypes(ctx context.Context) ([]domain.DictionaryItem, error) {
	return a.getDictionary(ctx, "deal_type")
}


func (a *FilterRepository) GetUniqueCitiesByRegion(ctx context.Context, region string) ([]string, error) {
    query := `
        SELECT DISTINCT city_or_district 
        FROM general_properties 
        WHERE region = $1 AND city_or_district != ''
        ORDER BY city_or_district
    `
    rows, err := a.pool.Query(ctx, query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique deal types: %w", err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var city string
		if err := rows.Scan(&city); err == nil {
			items = append(items, city)
		}
	}
	return items, rows.Err()
}


func (a *FilterRepository) GetTotalCount(ctx context.Context, req domain.FindObjectsFilters) (int, error) {
	joinClause, whereClause, args := applyFilters(req)

	query := fmt.Sprintf(`
		WITH latest_visible_objects AS (
			SELECT			
				ROW_NUMBER() OVER(PARTITION BY gp.master_object_id ORDER BY gp.updated_at DESC) as rn
			FROM
				general_properties gp
				%s
			%s -- Вставляем WHERE с фильтрами (region, category, etc.)
		)
		SELECT
			COUNT(*)
		FROM
			latest_visible_objects
		WHERE
			rn = 1;
	`, joinClause, whereClause)

	var count int
	err := a.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}
	return count, nil
}

func translator(field, systemName string) string {
	switch field {
	case "category":
		return translateCategory(systemName)
	
	case "deal_type":
		return translateDealType(systemName)

	default:
		return systemName
	}

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


func (r *FilterRepository) GetApartmentDistinctRooms(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "apartments", "rooms_amount")
}

func (r *FilterRepository) GetApartmentFloorsRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "apartments", "floor_number")
}

func (r *FilterRepository) GetApartmentBuildingFloorsRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "apartments", "building_floors")
}

func (r *FilterRepository) GetApartmentTotalAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "apartments", "total_area")
}

func (r *FilterRepository) GetApartmentLivingSpaceAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "apartments", "living_space_area")
}

func (r *FilterRepository) GetApartmentKitchenAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "apartments", "kitchen_area")
}

func (r *FilterRepository) GetApartmentYearBuiltRange(ctx context.Context) (*domain.RangeResult, error) {
    return r.getRange(ctx, "apartments", "year_built")
}

func (r *FilterRepository) GetApartmentDistinctWallMaterials(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "apartments", "wall_material")
}

func (r *FilterRepository) GetApartmentDistinctRepairStates(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "apartments", "repair_state")
}

func (r *FilterRepository) GetApartmentDistinctBathroomTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "apartments", "bathroom_type")
}

func (r *FilterRepository) GetApartmentDistinctBalconyTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "apartments", "balcony_type")
}




func (r *FilterRepository) GetHouseDistinctRooms(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "rooms_amount")
}

func (r *FilterRepository) GetHouseDistinctTypes(ctx context.Context) ([]interface{}, error) {
    return r.getDistinctValues(ctx, "houses", "house_type")
}

func (r *FilterRepository) GetHouseTotalAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "houses", "total_area")
}

func (r *FilterRepository) GetHouseLivingSpaceAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "houses", "living_space_area")
}

func (r *FilterRepository) GetHouseKitchenAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "houses", "kitchen_area")
}

func (r *FilterRepository) GetHousePlotAreaRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "houses", "plot_area")
}

func (r *FilterRepository) GetHouseFloorsRange(ctx context.Context) (*domain.RangeResult, error) {
	return r.getRange(ctx, "houses", "building_floors")
}

func (r *FilterRepository) GetHouseYearBuiltRange(ctx context.Context) (*domain.RangeResult, error) {
    return r.getRange(ctx, "houses", "year_built")
}

func (r *FilterRepository) GetHouseDistinctWallMaterials(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "wall_material")
}

func (r *FilterRepository) GetHouseDistinctRoofMaterials(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "roof_material")
}

func (r *FilterRepository) GetHouseDistinctWaterTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "water")
}

func (r *FilterRepository) GetHouseDistinctHeatingTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "heating")
}

func (r *FilterRepository) GetHouseDistinctElectricityTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "electricity")
}

func (r *FilterRepository) GetHouseDistinctSewageTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "sewage")
}

func (r *FilterRepository) GetHouseDistinctGazTypes(ctx context.Context) ([]interface{}, error) {
	return r.getDistinctValues(ctx, "houses", "gaz")
}

// // buildBaseQuery - хелпер для построения WHERE clause.
// func (a *FilterRepository) buildFilterQuery(req domain.FilterOptions) (string, string, []interface{}) {
// 	joinClause := ""
// 	conditions := []string{"gp.is_source_duplicate = false", "gp.status = 'active'"}
// 	args := make([]interface{}, 0)
// 	argId := 1

// 	// Динамически добавляем JOIN, если он нужен для фильтрации или выборки
// 	switch req.Category {
// 	case "apartment":
// 		joinClause = "JOIN apartments d ON gp.id = d.property_id"
// 	case "house":
// 		joinClause = "JOIN houses d ON gp.id = d.property_id"
// 	case "room":
// 		joinClause = "JOIN rooms d ON gp.id = d.property_id"
// 	// ... и так далее для других категорий с таблицами деталей
// 	}

// 	// Добавляем фильтры по `general_properties`
// 	if req.Category != "" {
// 		conditions = append(conditions, fmt.Sprintf("gp.category = $%d", argId))
// 		args = append(args, req.Category)
// 		argId++
// 	}
// 	if req.Region != "" {
// 		conditions = append(conditions, fmt.Sprintf("gp.region = $%d", argId))
// 		args = append(args, req.Region)
// 		argId++
// 	}
// 	if req.DealType != "" {
// 		conditions = append(conditions, fmt.Sprintf("gp.deal_type = $%d", argId))
// 		args = append(args, req.DealType)
// 		argId++
// 	}
	
// 	whereClause := ""
// 	if len(conditions) > 0 {
// 		whereClause = "WHERE " + strings.Join(conditions, " AND ")
// 	}

// 	return joinClause, whereClause, args
// }