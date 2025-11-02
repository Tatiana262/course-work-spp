package postgres

import (
	"context"
	"log"
	"fmt"
	"storage-service/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)



// BatchSave сохраняет пачку записей, используя протокол COPY для максимальной производительности
func (a *PostgresStorageAdapter) BatchSave(ctx context.Context, records []domain.RealEstateRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. подготовка хэшей
	hashes := make([]string, len(records))
	uniqueHashesSet := make(map[string]struct{})
	for i, rec := range records {
		payload := buildHashPayload(rec) 
		hash := calculateObjectHash(payload) 
		hashes[i] = hash
		uniqueHashesSet[hash] = struct{}{}
	}

	uniqueHashes := make([]string, 0, len(uniqueHashesSet))
	for hash := range uniqueHashesSet {
		uniqueHashes = append(uniqueHashes, hash)
	}

	// 2. создание всех недостающих master_objects атомарно и быстро
	_, err = tx.Exec(ctx,
		`INSERT INTO master_objects (canonical_hash)
		 SELECT unnest($1::varchar[])
		 ON CONFLICT (canonical_hash) DO NOTHING`,
		uniqueHashes,
	)
	if err != nil {
		return fmt.Errorf("failed to ensure master objects exist: %w", err)
	}

	// получение карты хэш - ID master_object
	hashToMasterIDMap := make(map[string]uuid.UUID)
	rows, err := tx.Query(ctx,
		`SELECT canonical_hash, id FROM master_objects WHERE canonical_hash = ANY($1)`,
		uniqueHashes,
	)
	if err != nil {
		return fmt.Errorf("failed to query master objects: %w", err)
	}
	for rows.Next() {
		var hash string
		var masterID uuid.UUID
		if err := rows.Scan(&hash, &masterID); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan master object: %w", err)
		}
		hashToMasterIDMap[hash] = masterID
	}
	rows.Close()


	// получение существующих дубликатов (master_object_id + source) для наших master_objects, чтобы правильно выставить флаг is_source_duplicate
	masterIDs := make([]uuid.UUID, 0, len(hashToMasterIDMap))
	for _, id := range hashToMasterIDMap {
		masterIDs = append(masterIDs, id)
	}

	existingSourceDuplicates := make(map[string]struct{}) // Ключ: "master_id|source"
	if len(masterIDs) > 0 {
		rows, err = tx.Query(ctx,
			`SELECT master_object_id, source FROM general_properties
			 WHERE master_object_id = ANY($1) AND is_source_duplicate = false`,
			masterIDs,
		)
		if err != nil {
			return fmt.Errorf("failed to query existing source duplicates: %w", err)
		}
		for rows.Next() {
			var masterID uuid.UUID
			var source string
			if err := rows.Scan(&masterID, &source); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan source duplicate: %w", err)
			}
			key := fmt.Sprintf("%s|%s", masterID, source)
			existingSourceDuplicates[key] = struct{}{}
		}
		rows.Close()
	}


	// 3. формирование данных для записи
	generalRows := make([][]interface{}, 0, len(records))
	tempIDToDetails := make(map[uuid.UUID]interface{})
	tempIDToSourceKey := make(map[uuid.UUID]string)

	// карта для обработки дубликатов внутри текущего пакета
	batchSourcesSeen := make(map[string]struct{}) // Ключ: master_object_id|source

	for i, rec := range records {
		dbGeneral := rec.General
		hash := hashes[i]
		masterID := hashToMasterIDMap[hash]

		// Логика определения "плохого" дубликата (с того же источника)
		isSourceDuplicate := false
		key := fmt.Sprintf("%s|%s", masterID, dbGeneral.Source)

		// 1. Проверяем в объектах, которые уже есть в БД, видели ли мы объект с master_object_id из этого источника
		if _, exists := existingSourceDuplicates[key]; exists {
			isSourceDuplicate = true
		}
		// 2. Проверяем в объектах, которые мы обработали ранее в этом же пакете
		if _, seen := batchSourcesSeen[key]; seen {
			isSourceDuplicate = true
		}

		// Если это первый объект из источника для этого master_object_id, сохраняем его
		if !isSourceDuplicate {
			batchSourcesSeen[key] = struct{}{}
		}

		// Подготовка данных для COPY
		tempIDToSourceKey[dbGeneral.ID] = fmt.Sprintf("%s|%d", rec.General.Source, rec.General.SourceAdID)
		tempIDToDetails[dbGeneral.ID] = rec.Details

		generalRows = append(generalRows, []interface{}{
			dbGeneral.ID, dbGeneral.Source, dbGeneral.SourceAdID, dbGeneral.CreatedAt, dbGeneral.UpdatedAt,
			dbGeneral.Category, dbGeneral.AdLink, dbGeneral.SaleType, dbGeneral.Currency,
			dbGeneral.Images,  dbGeneral.ListTime, dbGeneral.Description, dbGeneral.Title,  dbGeneral.DealType, 
			dbGeneral.Coordinates, dbGeneral.CityOrDistrict, dbGeneral.Region,
			dbGeneral.PriceBYN, dbGeneral.PriceUSD, dbGeneral.PriceEUR, dbGeneral.Address, dbGeneral.IsAgency, dbGeneral.SellerName,
			dbGeneral.SellerDetails,
			masterID,          
			isSourceDuplicate, 
			"active",
		})
	}


	// 4. Массовая запись в БД с временной таблицей из-за coordinates
	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE temp_general_properties (LIKE general_properties) ON COMMIT DROP;
	`)
	if err != nil {
		return fmt.Errorf("failed to create temp table for general_properties: %w", err)
	}

	// Меняем тип coordinates
	_, err = tx.Exec(ctx, `
		ALTER TABLE temp_general_properties ALTER COLUMN coordinates TYPE TEXT;
	`)
	if err != nil {
		return fmt.Errorf("failed to alter temp table column type: %w", err)
	}

	// Имена колонок для COPY 
	generalColumns := []string{
		"id", "source", "source_ad_id", "created_at", "updated_at", "category", "ad_link", "sale_type",
		"currency", "images", "list_time", "description", "title", "deal_type",
		"coordinates", "city_or_district", "region", "price_byn", "price_usd", "price_eur",
		"address", "is_agency", "seller_name", "seller_details",
		"master_object_id", "is_source_duplicate", "status",
	}

	// COPY во временную таблицу
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"temp_general_properties"},
		generalColumns,
		pgx.CopyFromRows(generalRows),
	)
	if err != nil {
		return fmt.Errorf("failed to copy to temp_general_properties: %w", err)
	}

	// атомарное обновление существующих записей и вставка новых.
	finalIDMap := make(map[string]uuid.UUID) // key: source|source_ad_id, value: final_id
	rows, err = tx.Query(ctx, `
		INSERT INTO general_properties (
			id, source, source_ad_id, created_at, updated_at, category, ad_link, sale_type,
			currency, images, list_time, description, title, deal_type,
			coordinates, city_or_district, region, price_byn, price_usd, price_eur, address, is_agency,
			seller_name, seller_details,
			master_object_id, is_source_duplicate
		)
		SELECT
			id, source, source_ad_id, created_at, updated_at, category, ad_link, sale_type,
			currency, images, list_time, description, title, deal_type,
			coordinates::geography, -- Преобразуем TEXT в GEOGRAPHY
			city_or_district, region, price_byn, price_usd, price_eur, address,
			is_agency, seller_name, seller_details,
			master_object_id, is_source_duplicate
		FROM temp_general_properties
		ON CONFLICT (source, source_ad_id) DO UPDATE SET
			updated_at = EXCLUDED.updated_at,
			list_time = EXCLUDED.list_time,
			price_byn = EXCLUDED.price_byn,
			price_usd = EXCLUDED.price_usd,
			price_eur = EXCLUDED.price_eur,
			description = EXCLUDED.description,
			title = EXCLUDED.title,
			images = EXCLUDED.images,
			master_object_id = EXCLUDED.master_object_id, -- Обновляем ссылку на мастера
			is_source_duplicate = EXCLUDED.is_source_duplicate -- Обновляем флаг дубликата
		RETURNING id, source, source_ad_id; -- Возвращаем id для связи с деталями
	`)
	if err != nil {
		return fmt.Errorf("failed to merge from temp_general_properties: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var finalID uuid.UUID
		var source string
		var sourceAdID int64
		if err := rows.Scan(&finalID, &source, &sourceAdID); err != nil {
			return fmt.Errorf("failed to scan returned id: %w", err)
		}
		key := fmt.Sprintf("%s|%d", source, sourceAdID)
		finalIDMap[key] = finalID
	}

	// 5. массовая запись деталей + соединение деталей с ID из БД
	apartmentDetails := make(map[uuid.UUID]*domain.Apartment)
	houseDetails := make(map[uuid.UUID]*domain.House)
	commercialDetails := make(map[uuid.UUID]*domain.Commercial)
	roomDetails := make(map[uuid.UUID]*domain.Room)
	garageAndParkingDetails := make(map[uuid.UUID]*domain.GarageAndParking)
	plotDetails := make(map[uuid.UUID]*domain.Plot)
	newBuildingDetails := make(map[uuid.UUID]*domain.NewBuilding)

	// Проход по карте с деталями, которую мы собрали в самом начале
	for tempID, genericDetails := range tempIDToDetails {
		// по временному id находится ключ source|source_ad_id
		sourceKey, ok := tempIDToSourceKey[tempID]
		if !ok {
			log.Printf("Consistency error: sourceKey not found for tempID %s", tempID)
			continue
		}

		// по ключу source|source_ad_id находится id, который вернула БД
		finalID, ok := finalIDMap[sourceKey]
		if !ok {
			log.Printf("Consistency error: finalID not found for sourceKey %s", sourceKey)
			continue
		}

		// type switch, чтобы положить детали в правильную, строго типизированную карту
		switch details := genericDetails.(type) {
		case *domain.Apartment:
			// связываем финальный id из БД с деталями квартиры
			apartmentDetails[finalID] = details
		case *domain.House:
			houseDetails[finalID] = details
		case *domain.Commercial:
			commercialDetails[finalID] = details
		case *domain.Room:
			roomDetails[finalID] = details
		case *domain.GarageAndParking:
			garageAndParkingDetails[finalID] = details
		case *domain.Plot:
			plotDetails[finalID] = details
		case *domain.NewBuilding:
			newBuildingDetails[finalID] = details
		}
	}

	// вызов соответствующих функций для массовой записи
	if len(apartmentDetails) > 0 {
		err = a.batchSaveApartmentDetails(ctx, tx, apartmentDetails)
		if err != nil {
			return fmt.Errorf("failed to batch save apartment details: %w", err)
		}
	}

	// if len(houseDetails) > 0 {
	// 	err = a.batchSaveHouseDetails(ctx, tx, houseDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save house details: %w", err)
	// 	}
	// }

	// if len(commercialDetails) > 0 {
	// 	err = a.batchSaveCommercialDetails(ctx, tx, commercialDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save commercial details: %w", err)
	// 	}
	// }

	// if len(roomDetails) > 0 {
	// 	err = a.batchSaveRoomDetails(ctx, tx, roomDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save room details: %w", err)
	// 	}
	// }

	// if len(garageAndParkingDetails) > 0 {
	// 	err = a.batchSaveGarageAndParkingDetails(ctx, tx, garageAndParkingDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save garage and parking details: %w", err)
	// 	}
	// }

	// if len(plotDetails) > 0 {
	// 	err = a.batchSavePlotDetails(ctx, tx, plotDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save plot details: %w", err)
	// 	}
	// }

	// if len(newBuildingDetails) > 0 {
	// 	err = a.batchSaveNewBuildingDetails(ctx, tx, newBuildingDetails)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to batch save new building details: %w", err)
	// 	}
	// }

	return tx.Commit(ctx)

}



// batchSaveApartmentDetails - сохранение деталей квартиры
func (a *PostgresStorageAdapter) batchSaveApartmentDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Apartment) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_apartments (LIKE apartments) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for apartments: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		// dbApt, err := toDBApartment(detail)
		if err != nil { return fmt.Errorf("failed to map apartment details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.RoomsAmount, detail.FloorNumber, detail.BuildingFloors, detail.TotalArea, detail.LivingSpaceArea, detail.KitchenArea,
			detail.YearBuilt, detail.WallMaterial, detail.RepairState, detail.BathroomType, detail.BalconyType,
			detail.PricePerSquareMeter, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "rooms_amount", "floor_number", "building_floors", "total_area", "living_space_area", "kitchen_area",
		"year_built", "wall_material", "repair_state", "bathroom_type", "balcony_type", "price_per_square_meter", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_apartments"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_apartments: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO apartments SELECT * FROM temp_apartments
		ON CONFLICT (property_id) DO UPDATE SET
			rooms_amount = EXCLUDED.rooms_amount,
			floor_number = EXCLUDED.floor_number,
			building_floors = EXCLUDED.building_floors,
			total_area = EXCLUDED.total_area,
			living_space_area = EXCLUDED.living_space_area,
			kitchen_area = EXCLUDED.kitchen_area,
			year_built = EXCLUDED.year_built,
			wall_material = EXCLUDED.wall_material,
			repair_state = EXCLUDED.repair_state,
			bathroom_type = EXCLUDED.bathroom_type,
			balcony_type = EXCLUDED.balcony_type,
			price_per_square_meter = EXCLUDED.price_per_square_meter,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_apartments: %w", err) }

	return nil
}


func (a *PostgresStorageAdapter) batchSaveHouseDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.House) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_houses (LIKE houses) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for houses: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map house details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.TotalArea, detail.PlotArea, detail.WallMaterial, detail.Condition, detail.YearBuilt,
			detail.LivingSpaceArea, detail.BuildingFloors, detail.RoomsAmount, detail.KitchenSize, detail.Electricity, detail.InGardeningCommunity,
			detail.Water, detail.Heating, detail.Sewage, detail.Gaz, detail.RoofMaterial, detail.ContractNumberAndDate, detail.HouseType, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "total_area", "plot_area", "wall_material", "condition", "year_built", 
			"living_space_area", "building_floors", "rooms_amount", "kitchen_size", "electricity",
			"in_gardening_community", "water", "heating", "sewage", "gaz", "roof_material", "contract_number_and_date", 
			"house_type", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_houses"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_houses: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO houses SELECT * FROM temp_houses
		ON CONFLICT (property_id) DO UPDATE SET
			total_area = EXCLUDED.total_area,
			plot_area = EXCLUDED.plot_area,
			wall_material = EXCLUDED.wall_material,
			condition = EXCLUDED.condition,
			year_built = EXCLUDED.year_built,
			living_space_area = EXCLUDED.living_space_area,
			building_floors = EXCLUDED.building_floors,
			rooms_amount = EXCLUDED.rooms_amount,
			kitchen_size = EXCLUDED.kitchen_size,
			electricity = EXCLUDED.electricity,
			in_gardening_community = EXCLUDED.in_gardening_community,
			water = EXCLUDED.water,
			heating = EXCLUDED.heating,
			sewage = EXCLUDED.sewage,
			gaz = EXCLUDED.gaz,
			roof_material = EXCLUDED.roof_material,
			contract_number_and_date = EXCLUDED.contract_number_and_date,
			house_type = EXCLUDED.house_type,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_houses: %w", err) }

	return nil
}


func (a *PostgresStorageAdapter) batchSaveCommercialDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Commercial) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_commercial (LIKE commercial) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for commercial: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map commercial details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.PropertyType, detail.Condition, detail.FloorNumber, detail.BuildingFloors,
			detail.TotalArea, detail.CommercialImprovements, detail.CommercialRepair, detail.IsPartlySellOrRent, detail.PricePerSquareMeter,
			detail.ContractNumberAndDate, detail.RoomsAmount, detail.CommercialBuildingLocation, detail.CommercialRentType, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "property_type", "condition", "floor_number", "building_floors", "total_area", 
			"commercial_improvements", "commercial_repair", "partly_sell_or_rent", "price_per_square_meter", "contract_number_and_date",
			"rooms_amount", "commercial_building_location", "commercial_rent_type", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_commercial"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_commercial: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO commercial SELECT * FROM temp_commercial
		ON CONFLICT (property_id) DO UPDATE SET
			property_type = EXCLUDED.property_type,
			condition = EXCLUDED.condition,
			floor_number = EXCLUDED.floor_number,
			building_floors = EXCLUDED.building_floors,
			total_area = EXCLUDED.total_area,
			commercial_improvements = EXCLUDED.commercial_improvements,
			commercial_repair = EXCLUDED.commercial_repair,
			partly_sell_or_rent = EXCLUDED.partly_sell_or_rent,
			price_per_square_meter = EXCLUDED.price_per_square_meter,
			contract_number_and_date = EXCLUDED.contract_number_and_date,
			rooms_amount = EXCLUDED.rooms_amount,
			commercial_building_location = EXCLUDED.commercial_building_location,
			commercial_rent_type = EXCLUDED.commercial_rent_type,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_commercial: %w", err) }

	return nil
}


func (a *PostgresStorageAdapter) batchSaveGarageAndParkingDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.GarageAndParking) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_garages_and_parkings (LIKE garages_and_parkings) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for garages_and_parkings: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map garage_and_parking details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.PropertyType, detail.ParkingPlacesAmount, detail.TotalArea, detail.Improvements,
			detail.Heating, detail.ParkingType, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "property_type", "parking_places_amount", "total_area", "improvements", "heating", "parking_type", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_garages_and_parkings"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_garages_and_parkings: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO garages_and_parkings SELECT * FROM temp_garages_and_parkings
		ON CONFLICT (property_id) DO UPDATE SET
			property_type = EXCLUDED.property_type,
			parking_places_amount = EXCLUDED.parking_places_amount,
			total_area = EXCLUDED.total_area,
			improvements = EXCLUDED.improvements,
			heating = EXCLUDED.heating,
			parking_type = EXCLUDED.parking_type,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_garages_and_parkings: %w", err) }

	return nil
}


func (a *PostgresStorageAdapter) batchSaveRoomDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Room) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_rooms (LIKE rooms) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for rooms: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map room details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.Condition, detail.Bathroom, detail.SuggestedRoomsAmount, detail.RoomsAmount, detail.FloorNumber, 
			detail.BuildingFloors, detail.TotalArea, detail.IsBalcony, detail.RentalType, detail.LivingSpaceArea, detail.FlatRepair, detail.IsFurniture,
			detail.KitchenSize, detail.KitchenItems, detail.BathItems, detail.FlatRentForWhom, detail.FlatWindowsSide, detail.YearBuilt, detail.WallMaterial,
			detail.FlatImprovement, detail.RoomType, detail.ContractNumberAndDate, detail.FlatBuildingImprovements, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "condition", "bathroom", "suggested_rooms_amount", "rooms_amount", "floor_number", "building_floors", "total_area", "is_balcony",
			"rental_type", "living_space_area", "flat_repair", "is_furniture", "kitchen_size", "kitchen_items", "bath_items", "flat_rent_for_whom", 
			"flat_windows_side", "year_built", "wall_material", "flat_improvement", "room_type", "contract_number_and_date", "flat_building_improvements",
			"parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_rooms"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_rooms: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO rooms SELECT * FROM temp_rooms
		ON CONFLICT (property_id) DO UPDATE SET
			condition = EXCLUDED.condition,
			bathroom = EXCLUDED.bathroom,
			suggested_rooms_amount = EXCLUDED.suggested_rooms_amount,
			rooms_amount = EXCLUDED.rooms_amount,
			floor_number = EXCLUDED.floor_number,
			building_floors = EXCLUDED.building_floors,
			total_area = EXCLUDED.total_area,
			is_balcony = EXCLUDED.is_balcony,
			rental_type = EXCLUDED.rental_type,
			living_space_area = EXCLUDED.living_space_area,
			flat_repair = EXCLUDED.flat_repair,
			is_furniture = EXCLUDED.is_furniture,
			kitchen_size = EXCLUDED.kitchen_size,
			kitchen_items = EXCLUDED.kitchen_items,
			bath_items = EXCLUDED.bath_items,
			flat_rent_for_whom = EXCLUDED.flat_rent_for_whom,
			flat_windows_side = EXCLUDED.flat_windows_side,
			year_built = EXCLUDED.year_built,
			wall_material = EXCLUDED.wall_material,
			flat_improvement = EXCLUDED.flat_improvement,
			room_type = EXCLUDED.room_type,
			contract_number_and_date = EXCLUDED.contract_number_and_date,
			flat_building_improvements = EXCLUDED.flat_building_improvements,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_rooms: %w", err) }

	return nil
}

func (a *PostgresStorageAdapter) batchSavePlotDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Plot) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_plots (LIKE plots) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for plots: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map plot details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.PlotArea, detail.InGardeningCommunity, detail.PropertyRights, detail.Electricity, 
			detail.Water, detail.Gaz, detail.Sewage, detail.IsOutbuildings, detail.OutbuildingsType, detail.ContractNumberAndDate, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "plot_area", "in_gardening_community", "property_rights", "electricity", "water", "gaz", "sewage", "is_outbuildings",
			"outbuildings_type", "contract_number_and_date", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_plots"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_plots: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO plots SELECT * FROM temp_plots
		ON CONFLICT (property_id) DO UPDATE SET
			plot_area = EXCLUDED.plot_area,
			in_gardening_community = EXCLUDED.in_gardening_community,
			property_rights = EXCLUDED.property_rights,
			electricity = EXCLUDED.electricity,
			water = EXCLUDED.water,
			gaz = EXCLUDED.gaz,
			sewage = EXCLUDED.sewage,
			is_outbuildings = EXCLUDED.is_outbuildings,
			outbuildings_type = EXCLUDED.outbuildings_type,
			contract_number_and_date = EXCLUDED.contract_number_and_date,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_plots: %w", err) }

	return nil
}


func (a *PostgresStorageAdapter) batchSaveNewBuildingDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.NewBuilding) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_new_buildings (LIKE new_buildings) ON COMMIT DROP;`)
	if err != nil { return fmt.Errorf("failed to create temp table for new_buildings: %w", err) }

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		
		if err != nil { return fmt.Errorf("failed to map new_building details: %w", err) }
		rows = append(rows, []interface{}{
			propID, detail.Deadline, detail.RoomOptions, detail.Builder, detail.ShareParticipation,
			detail.FloorOptions, detail.WallMaterial, detail.CeilingHeight, detail.LayoutOptions, detail.WithFinishing, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "deadline", "room_options", "builder", "share_participation", "floor_options", "wall_material", "flat_ceiling_height",
			"layout_options", "with_finishing", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_new_buildings"}, columns, pgx.CopyFromRows(rows))
	if err != nil { return fmt.Errorf("failed to copy to temp_new_buildings: %w", err) }

	_, err = tx.Exec(ctx, `
		INSERT INTO new_buildings SELECT * FROM temp_new_buildings
		ON CONFLICT (property_id) DO UPDATE SET
			deadline = EXCLUDED.deadline,
			room_options = EXCLUDED.room_options,
			builder = EXCLUDED.builder,
			share_participation = EXCLUDED.share_participation,
			floor_options = EXCLUDED.floor_options,
			wall_material = EXCLUDED.wall_material,
			flat_ceiling_height = EXCLUDED.flat_ceiling_height,
			layout_options = EXCLUDED.layout_options,
			with_finishing = EXCLUDED.with_finishing,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil { return fmt.Errorf("failed to merge from temp_new_buildings: %w", err) }

	return nil
}