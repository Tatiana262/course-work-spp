package postgres

import (
	"context"
	"fmt"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FindWithFilters ищет объекты по набору фильтров с пагинацией
func (a *PostgresStorageAdapter) FindWithFilters(ctx context.Context, filters domain.FindObjectsFilters, limit, offset int) (*domain.PaginatedResult, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "FindWithFilters",
		// "filters":   filters,
		"limit":     limit,
		"offset":    offset,
	})

	
	// Получаем части запроса от билдера
	joinClause, whereClause, args := applyFilters(filters)

	// Выполняем два запроса (один для COUNT, другой для данных) в транзакции
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Запрос для подсчета общего количества с фильтрами
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT gp.master_object_id) FROM general_properties gp %s %s", joinClause, whereClause)
	var totalCount int64 
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		repoLogger.Error("Failed to count objects with filters", err, port.Fields{"query": countQuery})
		return nil, fmt.Errorf("failed to count objects with filters: %w", err)
	}

	repoLogger.Info("Total objects found", port.Fields{"total_count": totalCount})

	// Если ничего не найдено, нет смысла делать второй запрос
	if totalCount == 0 {
		return &domain.PaginatedResult{
			Objects:      []domain.GeneralPropertyInfo{},
			TotalCount:   0,
			CurrentPage:  offset/limit + 1, // на какой странице находимся
			ItemsPerPage: limit,            // с какими параметрами
		}, nil
	}

	var dataQuery strings.Builder
	dataQuery.WriteString(`
		WITH filtered_ranked_properties AS (
			SELECT 
				gp.id, gp.source, gp.source_ad_id, gp.updated_at, gp.category, gp.deal_type, gp.ad_link, 
				gp.title, gp.address, gp.price_byn, gp.price_usd, gp.price_eur, gp.currency, gp.images, gp.status, gp.master_object_id,
				ROW_NUMBER() OVER(PARTITION BY gp.master_object_id ORDER BY gp.updated_at DESC) as rn
			FROM general_properties gp `)
	dataQuery.WriteString(joinClause) // Добавляем JOIN
	dataQuery.WriteString(" ")
	dataQuery.WriteString(whereClause) // Добавляем WHERE
	dataQuery.WriteString(`)
		SELECT id, source, source_ad_id, updated_at, category, deal_type, ad_link, 
			   title, address, price_byn, price_usd, price_eur, currency, images, status, master_object_id
		FROM filtered_ranked_properties
		WHERE rn = 1
		ORDER BY updated_at DESC, id ASC
	`)

	limitOffsetArgs := append(args, limit, offset)
	limitOffsetQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", dataQuery.String(), len(args)+1, len(args)+2)

	rows, err := tx.Query(ctx, limitOffsetQuery, limitOffsetArgs...)
	if err != nil {
		repoLogger.Error("Failed to find objects with filters", err, port.Fields{"query": dataQuery.String()})
		return nil, fmt.Errorf("failed to find objects with filters: %w", err)
	}
	defer rows.Close()

	objects := make([]domain.GeneralPropertyInfo, 0, limit)
	for rows.Next() {
		var obj domain.GeneralPropertyInfo
		if err := rows.Scan(
			&obj.ID, &obj.Source, &obj.SourceAdID, &obj.UpdatedAt, &obj.Category, &obj.DealType,
			&obj.AdLink, &obj.Title, &obj.Address, &obj.PriceBYN, &obj.PriceUSD, &obj.PriceEUR,
			&obj.Currency, &obj.Images, &obj.Status, &obj.MasterObjectID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan object: %w", err)
		}
		objects = append(objects, obj)
	}

	repoLogger.Info("Successfully found objects for page", port.Fields{"count": len(objects)})

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Формируем и возвращаем результат
	result := &domain.PaginatedResult{
		Objects:      objects,
		TotalCount:   int(totalCount),
		CurrentPage:  offset/limit + 1,
		ItemsPerPage: limit,
	}

	return result, nil
}


func (a *PostgresStorageAdapter) FindBestByMasterIDs(ctx context.Context, masterIDs []string) ([]domain.GeneralPropertyInfo, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component":       "PostgresStorageAdapter",
		"method":          "FindBestByMasterIDs",
		"master_id_count": len(masterIDs),
	})

	if len(masterIDs) == 0 {
		repoLogger.Debug("Received empty list of master IDs, returning empty result.", nil)
		return []domain.GeneralPropertyInfo{}, nil
	}

	repoLogger.Debug("Querying for best objects by master IDs.", nil)
	query := `
        WITH ranked_objects AS (
            SELECT
                id, source, source_ad_id, updated_at, category, deal_type, ad_link, title,
				address, price_byn, price_usd, price_eur, currency, images, status, master_object_id,
                ROW_NUMBER() OVER(
                    PARTITION BY master_object_id
                    ORDER BY
                        (status = 'active') DESC,
                        is_source_duplicate ASC,
                        updated_at DESC
                ) as rn
            FROM
                general_properties
            WHERE
                master_object_id = ANY($1)
        )
        SELECT id, source, source_ad_id, updated_at, category, deal_type, ad_link, 
		title, address, price_byn, price_usd, price_eur, currency, images, status, master_object_id
        FROM ranked_objects
        WHERE rn = 1`

	rows, err := a.pool.Query(ctx, query, masterIDs)
	if err != nil {
		repoLogger.Error("Failed to query for best objects", err, port.Fields{"query": query})
		return nil, fmt.Errorf("failed to find best objects by master ids: %w", err)
	}
	defer rows.Close()

	objects := make([]domain.GeneralPropertyInfo, 0, len(masterIDs))
	for rows.Next() {
		var obj domain.GeneralPropertyInfo

		if err := rows.Scan(&obj.ID, &obj.Source, &obj.SourceAdID, &obj.UpdatedAt, &obj.Category, &obj.DealType,
			&obj.AdLink, &obj.Title, &obj.Address, &obj.PriceBYN, &obj.PriceUSD, &obj.PriceEUR,
			&obj.Currency, &obj.Images, &obj.Status, &obj.MasterObjectID); err != nil {

			repoLogger.Error("Failed to scan best object row", err, nil)
			return nil, fmt.Errorf("failed to scan best object: %w", err)
		}
		objects = append(objects, obj)
	}

	if err := rows.Err(); err != nil {
		repoLogger.Error("Error during best objects rows iteration", err, nil)
		return nil, err
	}

	repoLogger.Info("Successfully found best objects", port.Fields{"found_count": len(objects)})
	return objects, nil
}

// FindByID находит полную информацию об объекте, включая детали
func (a *PostgresStorageAdapter) GetPropertyDetails(ctx context.Context, propertyID uuid.UUID) (*domain.PropertyDetailsView, error) {
	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component":   "PostgresStorageAdapter",
		"method":      "GetPropertyDetails",
		"property_id": propertyID,
	})

	repoLogger.Debug("Starting to get full property details.", nil)

	result := &domain.PropertyDetailsView{}

	// Получаем основное объявление
	repoLogger.Debug("Querying for main property.", nil)
	mainQuery := `SELECT master_object_id, id, source, source_ad_id, updated_at, created_at, category, ad_link, sale_type, currency, images, list_time,
						 description, title, deal_type, city_or_district, region, price_byn, price_usd, price_eur, address, is_agency, seller_name, seller_details,
	                     status
	              FROM general_properties WHERE id = $1`

	// Сканируем в структуру
	err := a.pool.QueryRow(ctx, mainQuery, propertyID).Scan(
		&result.MainProperty.MasterObjectID, &result.MainProperty.ID, &result.MainProperty.Source, &result.MainProperty.SourceAdID, &result.MainProperty.UpdatedAt,
		&result.MainProperty.CreatedAt, &result.MainProperty.Category, &result.MainProperty.AdLink, &result.MainProperty.SaleType, &result.MainProperty.Currency,
		&result.MainProperty.Images, &result.MainProperty.ListTime, &result.MainProperty.Description, &result.MainProperty.Title, &result.MainProperty.DealType,
		&result.MainProperty.CityOrDistrict, &result.MainProperty.Region, &result.MainProperty.PriceBYN, &result.MainProperty.PriceUSD, &result.MainProperty.PriceEUR,
		&result.MainProperty.Address, &result.MainProperty.IsAgency, &result.MainProperty.SellerName, &result.MainProperty.SellerDetails, &result.MainProperty.Status, 
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			repoLogger.Warn("Main property not found.", nil)
			return nil, fmt.Errorf("property with id %s not found", propertyID) // Конкретная ошибка
		}
		repoLogger.Error("Failed to get main property", err, port.Fields{"query": mainQuery})
		return nil, fmt.Errorf("failed to get main property: %w", err)
	}
	repoLogger.Info("Successfully found main property.", port.Fields{"category": result.MainProperty.Category})

	// Получаем детали в зависимости от категории 
	switch result.MainProperty.Category {
	case "apartment":
		var details domain.Apartment
		detailsQuery := `SELECT rooms_amount, floor_number, building_floors, total_area, living_space_area, kitchen_area, year_built,
							wall_material, repair_state, bathroom_type, balcony_type, price_per_square_meter, is_new_condition, parameters 
		                 FROM apartments WHERE property_id = $1`
		err := a.pool.QueryRow(ctx, detailsQuery, propertyID).Scan(
			&details.RoomsAmount, &details.FloorNumber, &details.BuildingFloors, &details.TotalArea,
			&details.LivingSpaceArea, &details.KitchenArea, &details.YearBuilt, &details.WallMaterial, &details.RepairState,
			&details.BathroomType, &details.BalconyType, &details.PricePerSquareMeter, &details.IsNewCondition, &details.Parameters,
		)
		if err != nil && err != pgx.ErrNoRows {
			repoLogger.Error("Failed to get apartment details", err, port.Fields{"query": detailsQuery})
			return nil, fmt.Errorf("failed to get apartment details: %w", err)
		}
		result.Details = &details

	case "house":
		var details domain.House
		detailsQuery := `SELECT total_area, plot_area, living_space_area, kitchen_area, year_built, building_floors, rooms_amount, 
							wall_material, roof_material, house_type, electricity, water, heating, sewage, gaz, completion_percent, is_new_condition, parameters 
		                 FROM houses WHERE property_id = $1`
		err := a.pool.QueryRow(ctx, detailsQuery, propertyID).Scan(
			&details.TotalArea, &details.PlotArea, &details.LivingSpaceArea, &details.KitchenArea, &details.YearBuilt, &details.BuildingFloors,
			&details.RoomsAmount, &details.WallMaterial, &details.RoofMaterial, &details.HouseType, &details.Electricity, &details.Water, &details.Heating,
			&details.Sewage, &details.Gaz, &details.CompletionPercent, &details.IsNewCondition, &details.Parameters,
		)
		if err != nil && err != pgx.ErrNoRows {
			repoLogger.Error("Failed to get house details", err, port.Fields{"query": detailsQuery})
			return nil, fmt.Errorf("failed to get house details: %w", err)
		}
		result.Details = &details

	case "commercial":
		var details domain.Commercial
		detailsQuery := `SELECT property_type, floor_number, building_floors, total_area, commercial_improvements, commercial_repair,
							price_per_square_meter, rooms_range, commercial_building_location, commercial_rent_type, is_new_condition, parameters 
		                 FROM commercial WHERE property_id = $1`
		err := a.pool.QueryRow(ctx, detailsQuery, propertyID).Scan(
			&details.PropertyType, &details.FloorNumber, &details.BuildingFloors, &details.TotalArea, &details.CommercialImprovements,
			&details.CommercialRepair, &details.PricePerSquareMeter, &details.RoomsRange, &details.CommercialBuildingLocation,
			&details.CommercialRentType, &details.IsNewCondition, &details.Parameters,
		)
		if err != nil && err != pgx.ErrNoRows {
			repoLogger.Error("Failed to get commercial details", err, port.Fields{"query": detailsQuery})
			return nil, fmt.Errorf("failed to get commercial details: %w", err)
		}
		result.Details = &details
	default:
		repoLogger.Warn("No details handler for category", port.Fields{"category": result.MainProperty.Category})
	}

	// Получаем все связанные предложения (дубликаты)
	repoLogger.Debug("Querying for related duplicate offers.", port.Fields{"master_object_id": result.MainProperty.MasterObjectID})
	relatedQuery := `SELECT id, source, ad_link, is_source_duplicate, deal_type
	                 FROM general_properties
	                 WHERE master_object_id = $1 AND id != $2 AND status = 'active' 
					 ORDER BY is_source_duplicate ASC`

	rows, err := a.pool.Query(ctx, relatedQuery, result.MainProperty.MasterObjectID, propertyID)
	if err != nil {
		repoLogger.Error("Failed to get related offers", err, port.Fields{"query": relatedQuery})
		return nil, fmt.Errorf("failed to get related offers: %w", err)
	}
	defer rows.Close()

	relatedOffers := make([]domain.DuplicatesInfo, 0)
	for rows.Next() {
		var offer domain.DuplicatesInfo
		if err := rows.Scan(
			&offer.ID, &offer.Source, &offer.AdLink, &offer.IsSourceDuplicate, &offer.DealType,
		); err != nil {
			repoLogger.Error("Failed to scan related offer row", err, nil)
			return nil, fmt.Errorf("failed to scan related offer: %w", err)
		}
		relatedOffers = append(relatedOffers, offer)
	}

	if err = rows.Err(); err != nil {
		repoLogger.Error("Error during related offers rows iteration", err, nil)
		return nil, err
	}

	result.RelatedOffers = relatedOffers
	repoLogger.Info("Successfully found related offers.", port.Fields{"count": len(relatedOffers)})

	return result, nil
}
