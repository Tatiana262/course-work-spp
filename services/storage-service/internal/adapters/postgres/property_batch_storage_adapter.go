package postgres

import (
	"context"
	"strings"
	"fmt"
	"storage-service/internal/contextkeys"
	"storage-service/internal/core/domain"
	"storage-service/internal/core/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BatchSave сохраняет пачку записей, используя протокол COPY для максимальной производительности
func (a *PostgresStorageAdapter) BatchSave(ctx context.Context, records []domain.RealEstateRecord) (*domain.BatchSaveStats, error) {

	logger := contextkeys.LoggerFromContext(ctx)
	repoLogger := logger.WithFields(port.Fields{
		"component": "PostgresStorageAdapter",
		"method":    "BatchSave",
		"record_count": len(records),
	})
	
	if len(records) == 0 {
		repoLogger.Debug("No records to save, returning empty stats.", nil)
		return &domain.BatchSaveStats{}, nil
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		repoLogger.Error("Failed to begin transaction", err, nil)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	stats := &domain.BatchSaveStats{}

	// разделение потоков
	recordsToUpsert := make([]domain.RealEstateRecord, 0, len(records))
	recordsToArchive := make([]domain.RealEstateRecord, 0)

	for _, rec := range records {
		if rec.General.Status == "archived" {
			recordsToArchive = append(recordsToArchive, rec)
		} else {
			recordsToUpsert = append(recordsToUpsert, rec)
		}
	}

	repoLogger.Info("Processing records", port.Fields{
		"to_upsert": len(recordsToUpsert),
		"to_archive": len(recordsToArchive),
	})

	if len(recordsToArchive) > 0 {
		// один UPDATE для всех архивных записей
		// находим записи по их уникальному ключу (source, source_ad_id) и меняем статус

		// Собираем ключи для UPDATE
		keys := make([][]interface{}, len(recordsToArchive))
		for i, rec := range recordsToArchive {
			keys[i] = []interface{}{rec.General.Source, rec.General.SourceAdID}
		}

		sql := `
			WITH old_data AS (
				SELECT gp.id, gp.status
				FROM general_properties gp
				JOIN (VALUES %s) AS vals(source, source_ad_id)
					ON gp.source = vals.source AND gp.source_ad_id = vals.source_ad_id
			)
			UPDATE general_properties gp
				SET status = 'archived', updated_at = NOW()
				FROM old_data od
				WHERE gp.id = od.id
				RETURNING 
					gp.master_object_id,
					gp.source,
					gp.deal_type,
					gp.is_source_duplicate,
					(od.status != gp.status) AS status_was_changed;
		`

		columnTypes := []string{"TEXT", "BIGINT"}
		// Генерируем плейсхолдеры с типами
		placeholders := buildValuesPlaceholders(columnTypes, len(keys))
		formattedSQL := fmt.Sprintf(sql, placeholders)
		flatArgs := flatten(keys)

		repoLogger.Debug("Executing batch archive.", nil)
		
		rows, err := tx.Query(ctx, formattedSQL, flatArgs...)
		if err != nil {
			rows.Close()
			repoLogger.Error("Failed to batch archive properties", err, port.Fields{"query": formattedSQL})
			return nil, fmt.Errorf("failed to batch archive properties and count results: %w", err)
		}
		
		// Собираем информацию о свергнутых канонических объектах
		demotedChampions := make(map[string]bool) // Ключ: "master_id|source|deal_type"

		for rows.Next() {
			var masterID uuid.UUID
			var source, dealType string
			var wasSourceDuplicate, statusWasChanged bool
			if err := rows.Scan(&masterID, &source, &dealType, &wasSourceDuplicate, &statusWasChanged); err != nil { 
				repoLogger.Error("Failed to scan", err, port.Fields{"query": formattedSQL})
				return nil, fmt.Errorf("failed to scan: %w", err)
			}
			
			if statusWasChanged {
				stats.Archived++
			} else {
				// Если статус не изменился, значит, просто обновили
				stats.Updated++
			}
			// Если мы заархивировали хороший дубликат, запоминаем его
			if !wasSourceDuplicate && statusWasChanged {
				key := fmt.Sprintf("%s|%s|%s", masterID, source, dealType)
				demotedChampions[key] = true
			}
		}
		rows.Close()
		
		// Если были свергнутые канонические объекты, ищем им замену
		if len(demotedChampions) > 0 {
			repoLogger.Debug("Demoted champions found, electing new ones.", port.Fields{"count": len(demotedChampions)})
            // Собираем уникальные master_id для поиска
            // Используем map[uuid.UUID]struct{} для автоматического обеспечения уникальности
            masterIDSet := make(map[uuid.UUID]struct{})
            for key := range demotedChampions {
                // Разбираем ключ "master_id|source|deal_type"
                parts := strings.Split(key, "|")
                if len(parts) > 0 {
                    // Парсим первую часть ключа как UUID
                    masterID, err := uuid.Parse(parts[0])
                    if err == nil {
                        masterIDSet[masterID] = struct{}{}
                    }
                }
            }

            // Преобразуем set в срез для передачи в SQL-запрос
            masterIDsToReElect := make([]uuid.UUID, 0, len(masterIDSet))
            for id := range masterIDSet {
                masterIDsToReElect = append(masterIDsToReElect, id)
            }

			// находим для каждой группы самого свежего активного плохого дубликаблика и делаем его хорошим
			updateNewChampionsSQL := `
				WITH new_champions AS (
					SELECT
						id,
						ROW_NUMBER() OVER(
							PARTITION BY master_object_id, source, deal_type 
							ORDER BY updated_at DESC
						) as rn
					FROM general_properties
					WHERE master_object_id = ANY($1)
					  AND is_source_duplicate = true -- Ищем среди плохих
					  AND status = 'active'
				)
				UPDATE general_properties gp
				SET is_source_duplicate = false, updated_at = NOW()
				FROM new_champions nc
				WHERE gp.id = nc.id AND nc.rn = 1;
			`
			// Выполняем этот запрос
			cmdTag, err := tx.Exec(ctx, updateNewChampionsSQL, masterIDsToReElect)
			if err != nil {
				repoLogger.Error("Failed to elect new champions", err, nil)
				return nil, fmt.Errorf("failed to elect new source duplicates: %w", err)
			}
            repoLogger.Debug("Successfully elected new champions.", port.Fields{"elected_count": cmdTag.RowsAffected()})
		}
		repoLogger.Info("Batch archive complete", port.Fields{
			"archived_now": stats.Archived,
			"updated_archive": stats.Updated,
		})
	}

	
	if len(recordsToUpsert) > 0 {

		// подготовка хэшей
		hashes := make([]string, len(recordsToUpsert))
		uniqueHashesSet := make(map[string]struct{})
		for i, rec := range recordsToUpsert {
			payload := buildHashPayload(rec)     // функция для создания отпечатка
			hash := calculateObjectHash(payload) // функция хэширования
			hashes[i] = hash
			uniqueHashesSet[hash] = struct{}{}
		}

		uniqueHashes := make([]string, 0, len(uniqueHashesSet))
		for hash := range uniqueHashesSet {
			uniqueHashes = append(uniqueHashes, hash)
		}

		// Одним запросом создаются все недостающие master_objects
		repoLogger.Debug("Ensuring master objects exist.", port.Fields{"unique_hashes_count": len(uniqueHashes)})
		_, err = tx.Exec(ctx,
			`INSERT INTO master_objects (canonical_hash)
			SELECT unnest($1::varchar[])
			ON CONFLICT (canonical_hash) DO NOTHING`,
			uniqueHashes,
		)
		if err != nil {
			repoLogger.Error("Failed to ensure master objects exist", err, nil)
			return nil, fmt.Errorf("failed to ensure master objects exist: %w", err)
		}

		// когда все master_objects гарантированно существуют, получаем полную карту хэш -> ID master_object
		repoLogger.Debug("Querying master object IDs.", nil)
		hashToMasterIDMap := make(map[string]uuid.UUID)
		rows, err := tx.Query(ctx,
			`SELECT canonical_hash, id FROM master_objects WHERE canonical_hash = ANY($1)`,
			uniqueHashes,
		)
		if err != nil {
			repoLogger.Error("Failed to query master object IDs", err, nil)
			return nil, fmt.Errorf("failed to query master objects: %w", err)
		}
		for rows.Next() {
			var hash string
			var masterID uuid.UUID
			if err := rows.Scan(&hash, &masterID); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan master object: %w", err)
			}
			hashToMasterIDMap[hash] = masterID
		}
		rows.Close()
		repoLogger.Debug("Successfully mapped hashes to master IDs.", port.Fields{"mapped_count": len(hashToMasterIDMap)})

		// получение существующих хороших дубликатов для master_objects, чтобы правильно выставить флаг is_source_duplicate
		repoLogger.Debug("Querying for existing source duplicates.", nil)

		masterIDs := make([]uuid.UUID, 0, len(hashToMasterIDMap))
		for _, id := range hashToMasterIDMap {
			masterIDs = append(masterIDs, id)
		}

		existingSourceDuplicates := make(map[string]struct{}) // Ключ: "master_id|source|deal_type"
		existingSourceKeys := make(map[string]struct{}) // Ключ: "source|source_ad_id"
		
		if len(masterIDs) > 0 {
			rows, err = tx.Query(ctx,
				`SELECT master_object_id, source, deal_type, source_ad_id FROM general_properties
				WHERE master_object_id = ANY($1) AND is_source_duplicate = false AND status = 'active'`,
				masterIDs,
			)
			if err != nil {
				repoLogger.Error("Failed to query existing source duplicates", err, nil)
				return nil, fmt.Errorf("failed to query existing source duplicates: %w", err)
			}
			for rows.Next() {
				var masterID uuid.UUID
				var source, dealType string
				var sourceAdID int64
				if err := rows.Scan(&masterID, &source, &dealType, &sourceAdID); err != nil {
					rows.Close()
					return nil, fmt.Errorf("failed to scan source duplicate: %w", err)
				}
				key := fmt.Sprintf("%s|%s|%s", masterID, source, dealType)
				existingSourceDuplicates[key] = struct{}{}

				sourceKey := fmt.Sprintf("%s|%d", source, sourceAdID)
        		existingSourceKeys[sourceKey] = struct{}{}
			}
			rows.Close()
		}
		repoLogger.Debug("Successfully queried existing source duplicates.", port.Fields{"found_count": len(existingSourceDuplicates)})

		// формирование данных для записи
		generalRows := make([][]interface{}, 0, len(recordsToUpsert))
		tempIDToDetails := make(map[uuid.UUID]interface{})
		tempIDToSourceKey := make(map[uuid.UUID]string)

		// map для обработки дубликатов внутри текущего пакета
		batchSourcesSeen := make(map[string]struct{}) // Ключ: "master_id|source|deal_type"

		for i, rec := range recordsToUpsert {
			dbGeneral := rec.General
			hash := hashes[i]
			masterID := hashToMasterIDMap[hash]

			// Логика определения плохого дубликата
			isSourceDuplicate := false
			key := fmt.Sprintf("%s|%s|%s", masterID, dbGeneral.Source, dbGeneral.DealType)

			// Проверяем в объектах, которые уже есть в БД
			if _, exists := existingSourceDuplicates[key]; exists {
				currentSourceKey := fmt.Sprintf("%s|%d", dbGeneral.Source, dbGeneral.SourceAdID)
				if _, isOurself := existingSourceKeys[currentSourceKey]; !isOurself {
					// Хороший дубликат в базе есть и это не мы, значит мы - плохой дубликат
					isSourceDuplicate = true
				}
			}
			// Проверяем в объектах, которые мы обработали ранее в этом же пакете
			if _, seen := batchSourcesSeen[key]; seen {
				isSourceDuplicate = true
			}

			// Если это первый объект из источника для ключа master_id|source|deal_type в этой сессии, запоминаем его
			if !isSourceDuplicate {
				batchSourcesSeen[key] = struct{}{}
			}

			// Подготовка данных для COPY
			tempIDToSourceKey[dbGeneral.ID] = fmt.Sprintf("%s|%d", rec.General.Source, rec.General.SourceAdID)
			tempIDToDetails[dbGeneral.ID] = rec.Details

			generalRows = append(generalRows, []interface{}{
				dbGeneral.ID, dbGeneral.Source, dbGeneral.SourceAdID, dbGeneral.CreatedAt, dbGeneral.UpdatedAt,
				dbGeneral.Category, dbGeneral.AdLink, dbGeneral.SaleType, dbGeneral.Currency,
				dbGeneral.Images, dbGeneral.ListTime, dbGeneral.Description, dbGeneral.Title, dbGeneral.DealType,
				dbGeneral.Coordinates, dbGeneral.CityOrDistrict, dbGeneral.Region,
				dbGeneral.PriceBYN, dbGeneral.PriceUSD, dbGeneral.PriceEUR, dbGeneral.Address, dbGeneral.IsAgency, dbGeneral.SellerName,
				dbGeneral.SellerDetails,
				masterID,         
				isSourceDuplicate, 
				dbGeneral.Status,
			})

		}


		// массовая запись в БД (с использованием TEMP TABLE, в которой поле coordinates с типом TEXT)
		repoLogger.Debug("Creating temp table for general properties.", nil)
		_, err = tx.Exec(ctx, `
			CREATE TEMP TABLE temp_general_properties (LIKE general_properties) ON COMMIT DROP;
		`)
		if err != nil {
			repoLogger.Error("Failed to create temp table", err, nil)
			return nil, fmt.Errorf("failed to create temp table for general_properties: %w", err)
		}

		// Меняем тип coordinates
		_, err = tx.Exec(ctx, `
			ALTER TABLE temp_general_properties ALTER COLUMN coordinates TYPE TEXT;
		`)
		if err != nil {
			repoLogger.Error("Failed to alter temp table column coordinates type", err, nil)
			return nil, fmt.Errorf("failed to alter temp table column type: %w", err)
		}

		// Имена колонок для COPY
		generalColumns := []string{
			"id", "source", "source_ad_id", "created_at", "updated_at", "category", "ad_link", "sale_type",
			"currency", "images", "list_time", "description", "title", "deal_type",
			"coordinates", "city_or_district", "region", "price_byn", "price_usd", "price_eur",
			"address", "is_agency", "seller_name", "seller_details",
			"master_object_id", "is_source_duplicate", "status",
		}

		// Выполняем COPY во временную таблицу
		repoLogger.Debug("Copying data to temp table.", port.Fields{"rows_to_copy": len(generalRows)})
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"temp_general_properties"},
			generalColumns,
			pgx.CopyFromRows(generalRows),
		)
		if err != nil {
			repoLogger.Error("Failed to COPY data to temp table", err, nil)
			return nil, fmt.Errorf("failed to copy to temp_general_properties: %w", err)
		}


		// INSERT ... ON CONFLICT из временной таблицы в основную
		repoLogger.Debug("Merging data from temp table into main table.", nil)
		finalIDMap := make(map[string]uuid.UUID) // key: "source|source_ad_id", value: final_id

		rows, err = tx.Query(ctx, `
			INSERT INTO general_properties (
				id, source, source_ad_id, created_at, updated_at, category, ad_link, sale_type,
				currency, images, list_time, description, title, deal_type,
				coordinates, city_or_district, region, price_byn, price_usd, price_eur, address, is_agency,
				seller_name, seller_details,
				master_object_id, is_source_duplicate, status
			)
			SELECT
				id, source, source_ad_id, created_at, updated_at, category, ad_link, sale_type,
				currency, images, list_time, description, title, deal_type,
				coordinates::geography, -- Преобразуем TEXT в GEOGRAPHY
				city_or_district, region, price_byn, price_usd, price_eur, address,
				is_agency, seller_name, seller_details,
				master_object_id, is_source_duplicate, status
			FROM temp_general_properties
			ON CONFLICT (source, source_ad_id) DO UPDATE SET
				updated_at = EXCLUDED.updated_at,
				status = EXCLUDED.status,

				list_time = EXCLUDED.list_time, 
				price_byn = EXCLUDED.price_byn, 
				price_usd = EXCLUDED.price_usd, 
				price_eur = EXCLUDED.price_eur, 
				description = EXCLUDED.description, 
				title = EXCLUDED.title, 
				images = EXCLUDED.images,
				
				is_source_duplicate = EXCLUDED.is_source_duplicate
			RETURNING id, source, source_ad_id, (xmax = 0) AS inserted, status; -- Возвращаем id для связи с деталями
		`)
		if err != nil {
			repoLogger.Error("Failed to merge from temp table", err, nil)
			return nil, fmt.Errorf("failed to merge from temp_general_properties: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var finalID uuid.UUID
			var source string
			var sourceAdID int64
			var inserted bool // xmax = 0 означает, что строка была вставлена (INSERT)
			var status string
			if err := rows.Scan(&finalID, &source, &sourceAdID, &inserted, &status); err != nil {
				return nil, fmt.Errorf("failed to scan returned id: %w", err)
			}

			if inserted {
				stats.Created++
			} else {
				stats.Updated++
			}

			key := fmt.Sprintf("%s|%d", source, sourceAdID)
			finalIDMap[key] = finalID
		}
		repoLogger.Debug("Merge complete.", port.Fields{"created": stats.Created, "updated": stats.Updated})

		// массовая запись деталей
		// tempIDToDetails: map[временный_ID] -> {детали}
		// tempIDToSourceKey: map[временный_ID] -> "kufar|12345"
		// finalIDMap: map["kufar|12345"] -> постоянный_ID_из_БД
		// надо соединить {детали} с постоянным_ID_из_БД.

		// Инициализируем map, которые будут переданы в функции пакетного сохранения деталей
		// Ключом будет уже ID из general_properties
		repoLogger.Debug("Preparing details for batch save.", nil)

		apartmentDetails := make(map[uuid.UUID]*domain.Apartment)
		houseDetails := make(map[uuid.UUID]*domain.House)
		commercialDetails := make(map[uuid.UUID]*domain.Commercial)
		roomDetails := make(map[uuid.UUID]*domain.Room)
		garageAndParkingDetails := make(map[uuid.UUID]*domain.GarageAndParking)
		plotDetails := make(map[uuid.UUID]*domain.Plot)
		newBuildingDetails := make(map[uuid.UUID]*domain.NewBuilding)

		// Проходим по карте с деталями
		for tempID, genericDetails := range tempIDToDetails {
			// По временному ID находим бизнес-ключ ("kufar|12345")
			sourceKey, ok := tempIDToSourceKey[tempID]
			if !ok {
				repoLogger.Warn("Consistency error: sourceKey not found for tempID", port.Fields{
					"temp_id": tempID,
					"source_key": sourceKey,
				})
				continue
			}

			// По бизнес-ключу находим постоянный ID, который вернула БД
			finalID, ok := finalIDMap[sourceKey]
			if !ok {
				repoLogger.Warn("Consistency error: finalID not found for sourceKey", port.Fields{
					"source_key": sourceKey,
				})
				continue
			}

			// type switch, чтобы положить детали в правильную, строго типизированную карту
			switch details := genericDetails.(type) {
			case *domain.Apartment:
				// связываем финальный ID из БД с деталями квартиры
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

		// вызываем соответствующие функции для массовой записи.
		if len(apartmentDetails) > 0 {
			repoLogger.Debug("Batch saving apartment details.", port.Fields{"count": len(apartmentDetails)})
			err = a.batchSaveApartmentDetails(ctx, tx, apartmentDetails)
			if err != nil {
				return nil, fmt.Errorf("failed to batch save apartment details: %w", err)
			}
		}

		if len(houseDetails) > 0 {
			repoLogger.Debug("Batch saving house details.", port.Fields{"count": len(houseDetails)})
			err = a.batchSaveHouseDetails(ctx, tx, houseDetails)
			if err != nil {
				return nil, fmt.Errorf("failed to batch save house details: %w", err)
			}
		}

		if len(commercialDetails) > 0 {
			repoLogger.Debug("Batch saving commercial details.", port.Fields{"count": len(commercialDetails)})
			err = a.batchSaveCommercialDetails(ctx, tx, commercialDetails)
			if err != nil {
				return nil, fmt.Errorf("failed to batch save commercial details: %w", err)
			}
		}

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

		repoLogger.Info("Batch save complete", port.Fields{
			"processed_objects": len(recordsToUpsert),
		})

	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return stats, nil

}

// flatten преобразует срез срезов [][]interface{} в один плоский срез []interface{}
func flatten(data [][]interface{}) []interface{} {
	if len(data) == 0 {
		return nil
	}

	// Общая длина плоского среза будет равна (количество строк * количество колонок)
	flatSize := len(data) * len(data[0])
	flat := make([]interface{}, 0, flatSize)

	for _, row := range data {
		flat = append(flat, row...)
	}

	return flat
}


// buildValuesPlaceholders генерирует строку плейсхолдеров с явным приведением типов
// Например, для 2 строк с типами ["TEXT", "BIGINT"] он вернет "($1::TEXT, $2::BIGINT), ($3::TEXT, $4::BIGINT)"
func buildValuesPlaceholders(types []string, rows int) string {
	if rows == 0 || len(types) == 0 {
		return ""
	}
	columns := len(types)

	// срез для хранения отдельных групп
	rowPlaceholders := make([]string, rows)
	paramIndex := 1 // плейсхолдеры начинаются с $1

	for i := 0; i < rows; i++ {
		// срез для плейсхолдеров одной строки ["$1::TEXT", "$2::BIGINT"]
		colPlaceholders := make([]string, columns)
		for j := 0; j < columns; j++ {
			// Добавляем приведение типа к плейсхолдеру
			colPlaceholders[j] = fmt.Sprintf("$%d::%s", paramIndex, types[j])
			paramIndex++
		}
		// Объединяем плейсхолдеры строки в одну группу: "($1::TEXT, $2::BIGINT)"
		rowPlaceholders[i] = fmt.Sprintf("(%s)", strings.Join(colPlaceholders, ", "))
	}

	// Объединяем все группы в финальную строку
	return strings.Join(rowPlaceholders, ", ")
}



// batchSaveApartmentDetails - пакетная вставки для таблицы деталей
func (a *PostgresStorageAdapter) batchSaveApartmentDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Apartment) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_apartments (LIKE apartments) ON COMMIT DROP;`)
	if err != nil {
		return fmt.Errorf("failed to create temp table for apartments: %w", err)
	}

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		// dbApt, err := toDBApartment(detail)
		rows = append(rows, []interface{}{
			propID, detail.RoomsAmount, detail.FloorNumber, detail.BuildingFloors, detail.TotalArea, detail.LivingSpaceArea, detail.KitchenArea,
			detail.YearBuilt, detail.WallMaterial, detail.RepairState, detail.BathroomType, detail.BalconyType,
			detail.PricePerSquareMeter, detail.IsNewCondition, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "rooms_amount", "floor_number", "building_floors", "total_area", "living_space_area", "kitchen_area",
		"year_built", "wall_material", "repair_state", "bathroom_type", "balcony_type", "price_per_square_meter", "is_new_condition", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_apartments"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("failed to copy to temp_apartments: %w", err)
	}

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
			is_new_condition = EXCLUDED.is_new_condition,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil {
		return fmt.Errorf("failed to merge from temp_apartments: %w", err)
	}

	return nil
}

func (a *PostgresStorageAdapter) batchSaveHouseDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.House) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_houses (LIKE houses) ON COMMIT DROP;`)
	if err != nil {
		return fmt.Errorf("failed to create temp table for houses: %w", err)
	}

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		rows = append(rows, []interface{}{
			propID, detail.TotalArea, detail.PlotArea, detail.WallMaterial, detail.YearBuilt,
			detail.LivingSpaceArea, detail.BuildingFloors, detail.RoomsAmount, detail.KitchenArea, detail.Electricity,
			detail.Water, detail.Heating, detail.Sewage, detail.Gaz, detail.RoofMaterial, detail.HouseType, detail.IsNewCondition, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "total_area", "plot_area", "wall_material", "year_built",
		"living_space_area", "building_floors", "rooms_amount", "kitchen_area", "electricity",
		"water", "heating", "sewage", "gaz", "roof_material", 
		"house_type", "is_new_condition", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_houses"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("failed to copy to temp_houses: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO houses SELECT * FROM temp_houses
		ON CONFLICT (property_id) DO UPDATE SET
			total_area = EXCLUDED.total_area,
			plot_area = EXCLUDED.plot_area,
			wall_material = EXCLUDED.wall_material,
			year_built = EXCLUDED.year_built,
			living_space_area = EXCLUDED.living_space_area,
			building_floors = EXCLUDED.building_floors,
			rooms_amount = EXCLUDED.rooms_amount,
			kitchen_area = EXCLUDED.kitchen_area,
			electricity = EXCLUDED.electricity,
			water = EXCLUDED.water,
			heating = EXCLUDED.heating,
			sewage = EXCLUDED.sewage,
			gaz = EXCLUDED.gaz,
			roof_material = EXCLUDED.roof_material,
			house_type = EXCLUDED.house_type,
			is_new_condition = EXCLUDED.is_new_condition,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil {
		return fmt.Errorf("failed to merge from temp_houses: %w", err)
	}

	return nil
}

func (a *PostgresStorageAdapter) batchSaveCommercialDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Commercial) error {
	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_commercial (LIKE commercial) ON COMMIT DROP;`)
	if err != nil {
		return fmt.Errorf("failed to create temp table for commercial: %w", err)
	}

	rows := make([][]interface{}, 0, len(details))
	for propID, detail := range details {
		rows = append(rows, []interface{}{
			propID, detail.IsNewCondition, detail.PropertyType, detail.FloorNumber, detail.BuildingFloors,
			detail.TotalArea, detail.CommercialImprovements, detail.CommercialRepair, detail.PricePerSquareMeter, detail.RoomsRange,
			detail.CommercialBuildingLocation, detail.CommercialRentType, detail.Parameters,
		})
	}

	columns := []string{
		"property_id", "is_new_condition", "property_type", "floor_number", "building_floors", "total_area",
		"commercial_improvements", "commercial_repair", "price_per_square_meter", "rooms_range",
		"commercial_building_location", "commercial_rent_type", "parameters",
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_commercial"}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("failed to copy to temp_commercial: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO commercial SELECT * FROM temp_commercial
		ON CONFLICT (property_id) DO UPDATE SET
			is_new_condition = EXCLUDED.is_new_condition,
			property_type = EXCLUDED.property_type,
			floor_number = EXCLUDED.floor_number,
			building_floors = EXCLUDED.building_floors,
			total_area = EXCLUDED.total_area,
			commercial_improvements = EXCLUDED.commercial_improvements,
			commercial_repair = EXCLUDED.commercial_repair,
			price_per_square_meter = EXCLUDED.price_per_square_meter,
			rooms_range = EXCLUDED.rooms_range,
			commercial_building_location = EXCLUDED.commercial_building_location,
			commercial_rent_type = EXCLUDED.commercial_rent_type,
			parameters = EXCLUDED.parameters;
	`)
	if err != nil {
		return fmt.Errorf("failed to merge from temp_commercial: %w", err)
	}

	return nil
}

// func (a *PostgresStorageAdapter) batchSaveGarageAndParkingDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.GarageAndParking) error {
// 	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_garages_and_parkings (LIKE garages_and_parkings) ON COMMIT DROP;`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create temp table for garages_and_parkings: %w", err)
// 	}

// 	rows := make([][]interface{}, 0, len(details))
// 	for propID, detail := range details {
		
// 		rows = append(rows, []interface{}{
// 			propID, detail.PropertyType, detail.ParkingPlacesAmount, detail.TotalArea, detail.Improvements,
// 			detail.Heating, detail.ParkingType, detail.Parameters,
// 		})
// 	}

// 	columns := []string{
// 		"property_id", "property_type", "parking_places_amount", "total_area", "improvements", "heating", "parking_type", "parameters",
// 	}

// 	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_garages_and_parkings"}, columns, pgx.CopyFromRows(rows))
// 	if err != nil {
// 		return fmt.Errorf("failed to copy to temp_garages_and_parkings: %w", err)
// 	}

// 	_, err = tx.Exec(ctx, `
// 		INSERT INTO garages_and_parkings SELECT * FROM temp_garages_and_parkings
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			property_type = EXCLUDED.property_type,
// 			parking_places_amount = EXCLUDED.parking_places_amount,
// 			total_area = EXCLUDED.total_area,
// 			improvements = EXCLUDED.improvements,
// 			heating = EXCLUDED.heating,
// 			parking_type = EXCLUDED.parking_type,
// 			parameters = EXCLUDED.parameters;
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to merge from temp_garages_and_parkings: %w", err)
// 	}

// 	return nil
// }

// func (a *PostgresStorageAdapter) batchSaveRoomDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Room) error {
// 	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_rooms (LIKE rooms) ON COMMIT DROP;`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create temp table for rooms: %w", err)
// 	}

// 	rows := make([][]interface{}, 0, len(details))
// 	for propID, detail := range details {

// 		rows = append(rows, []interface{}{
// 			propID, detail.Condition, detail.Bathroom, detail.SuggestedRoomsAmount, detail.RoomsAmount, detail.FloorNumber,
// 			detail.BuildingFloors, detail.TotalArea, detail.IsBalcony, detail.RentalType, detail.LivingSpaceArea, detail.FlatRepair, detail.IsFurniture,
// 			detail.KitchenSize, detail.KitchenItems, detail.BathItems, detail.FlatRentForWhom, detail.FlatWindowsSide, detail.YearBuilt, detail.WallMaterial,
// 			detail.FlatImprovement, detail.RoomType, detail.ContractNumberAndDate, detail.FlatBuildingImprovements, detail.Parameters,
// 		})
// 	}

// 	columns := []string{
// 		"property_id", "condition", "bathroom", "suggested_rooms_amount", "rooms_amount", "floor_number", "building_floors", "total_area", "is_balcony",
// 		"rental_type", "living_space_area", "flat_repair", "is_furniture", "kitchen_size", "kitchen_items", "bath_items", "flat_rent_for_whom",
// 		"flat_windows_side", "year_built", "wall_material", "flat_improvement", "room_type", "contract_number_and_date", "flat_building_improvements",
// 		"parameters",
// 	}

// 	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_rooms"}, columns, pgx.CopyFromRows(rows))
// 	if err != nil {
// 		return fmt.Errorf("failed to copy to temp_rooms: %w", err)
// 	}

// 	_, err = tx.Exec(ctx, `
// 		INSERT INTO rooms SELECT * FROM temp_rooms
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			condition = EXCLUDED.condition,
// 			bathroom = EXCLUDED.bathroom,
// 			suggested_rooms_amount = EXCLUDED.suggested_rooms_amount,
// 			rooms_amount = EXCLUDED.rooms_amount,
// 			floor_number = EXCLUDED.floor_number,
// 			building_floors = EXCLUDED.building_floors,
// 			total_area = EXCLUDED.total_area,
// 			is_balcony = EXCLUDED.is_balcony,
// 			rental_type = EXCLUDED.rental_type,
// 			living_space_area = EXCLUDED.living_space_area,
// 			flat_repair = EXCLUDED.flat_repair,
// 			is_furniture = EXCLUDED.is_furniture,
// 			kitchen_size = EXCLUDED.kitchen_size,
// 			kitchen_items = EXCLUDED.kitchen_items,
// 			bath_items = EXCLUDED.bath_items,
// 			flat_rent_for_whom = EXCLUDED.flat_rent_for_whom,
// 			flat_windows_side = EXCLUDED.flat_windows_side,
// 			year_built = EXCLUDED.year_built,
// 			wall_material = EXCLUDED.wall_material,
// 			flat_improvement = EXCLUDED.flat_improvement,
// 			room_type = EXCLUDED.room_type,
// 			contract_number_and_date = EXCLUDED.contract_number_and_date,
// 			flat_building_improvements = EXCLUDED.flat_building_improvements,
// 			parameters = EXCLUDED.parameters;
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to merge from temp_rooms: %w", err)
// 	}

// 	return nil
// }

// func (a *PostgresStorageAdapter) batchSavePlotDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.Plot) error {
// 	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_plots (LIKE plots) ON COMMIT DROP;`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create temp table for plots: %w", err)
// 	}

// 	rows := make([][]interface{}, 0, len(details))
// 	for propID, detail := range details {

// 		rows = append(rows, []interface{}{
// 			propID, detail.PlotArea, detail.InGardeningCommunity, detail.PropertyRights, detail.Electricity,
// 			detail.Water, detail.Gaz, detail.Sewage, detail.IsOutbuildings, detail.OutbuildingsType, detail.ContractNumberAndDate, detail.Parameters,
// 		})
// 	}

// 	columns := []string{
// 		"property_id", "plot_area", "in_gardening_community", "property_rights", "electricity", "water", "gaz", "sewage", "is_outbuildings",
// 		"outbuildings_type", "contract_number_and_date", "parameters",
// 	}

// 	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_plots"}, columns, pgx.CopyFromRows(rows))
// 	if err != nil {
// 		return fmt.Errorf("failed to copy to temp_plots: %w", err)
// 	}

// 	_, err = tx.Exec(ctx, `
// 		INSERT INTO plots SELECT * FROM temp_plots
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			plot_area = EXCLUDED.plot_area,
// 			in_gardening_community = EXCLUDED.in_gardening_community,
// 			property_rights = EXCLUDED.property_rights,
// 			electricity = EXCLUDED.electricity,
// 			water = EXCLUDED.water,
// 			gaz = EXCLUDED.gaz,
// 			sewage = EXCLUDED.sewage,
// 			is_outbuildings = EXCLUDED.is_outbuildings,
// 			outbuildings_type = EXCLUDED.outbuildings_type,
// 			contract_number_and_date = EXCLUDED.contract_number_and_date,
// 			parameters = EXCLUDED.parameters;
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to merge from temp_plots: %w", err)
// 	}

// 	return nil
// }

// func (a *PostgresStorageAdapter) batchSaveNewBuildingDetails(ctx context.Context, tx pgx.Tx, details map[uuid.UUID]*domain.NewBuilding) error {
// 	_, err := tx.Exec(ctx, `CREATE TEMP TABLE temp_new_buildings (LIKE new_buildings) ON COMMIT DROP;`)
// 	if err != nil {
// 		return fmt.Errorf("failed to create temp table for new_buildings: %w", err)
// 	}

// 	rows := make([][]interface{}, 0, len(details))
// 	for propID, detail := range details {

// 		rows = append(rows, []interface{}{
// 			propID, detail.Deadline, detail.RoomOptions, detail.Builder, detail.ShareParticipation,
// 			detail.FloorOptions, detail.WallMaterial, detail.CeilingHeight, detail.LayoutOptions, detail.WithFinishing, detail.Parameters,
// 		})
// 	}

// 	columns := []string{
// 		"property_id", "deadline", "room_options", "builder", "share_participation", "floor_options", "wall_material", "flat_ceiling_height",
// 		"layout_options", "with_finishing", "parameters",
// 	}

// 	_, err = tx.CopyFrom(ctx, pgx.Identifier{"temp_new_buildings"}, columns, pgx.CopyFromRows(rows))
// 	if err != nil {
// 		return fmt.Errorf("failed to copy to temp_new_buildings: %w", err)
// 	}

// 	_, err = tx.Exec(ctx, `
// 		INSERT INTO new_buildings SELECT * FROM temp_new_buildings
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			deadline = EXCLUDED.deadline,
// 			room_options = EXCLUDED.room_options,
// 			builder = EXCLUDED.builder,
// 			share_participation = EXCLUDED.share_participation,
// 			floor_options = EXCLUDED.floor_options,
// 			wall_material = EXCLUDED.wall_material,
// 			flat_ceiling_height = EXCLUDED.flat_ceiling_height,
// 			layout_options = EXCLUDED.layout_options,
// 			with_finishing = EXCLUDED.with_finishing,
// 			parameters = EXCLUDED.parameters;
// 	`)
// 	if err != nil {
// 		return fmt.Errorf("failed to merge from temp_new_buildings: %w", err)
// 	}

// 	return nil
// }
