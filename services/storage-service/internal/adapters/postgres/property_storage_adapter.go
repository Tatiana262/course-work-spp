package postgres

import (
	"context"
	// "encoding/binary"
	// "encoding/json"
	"fmt"
	// "log"
	// "os"
	"storage-service/internal/core/domain"
	// "time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	// "github.com/twpayne/go-geom"
	// "github.com/twpayne/go-geom/encoding/wkb"
)

// PostgresStorageAdapter реализует PropertyStoragePort для PostgreSQL.
type PostgresStorageAdapter struct {
	pool *pgxpool.Pool
}

// NewPostgresStorageAdapter создает новый экземпляр адаптера.
func NewPostgresStorageAdapter(pool *pgxpool.Pool) (*PostgresStorageAdapter, error) {
	if pool == nil {
		return nil, fmt.Errorf("pgxpool.Pool cannot be nil")
	}
	return &PostgresStorageAdapter{
		pool: pool,
	}, nil
}

// Save сохраняет одну запись RealEstateRecord в базу данных в рамках одной транзакции.
func (a *PostgresStorageAdapter) Save(ctx context.Context, record domain.RealEstateRecord) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- Шаг 1: Преобразуем и вставляем основную запись ---
	// dbGeneral := record.General

	// sqlGeneral := `
	// 	INSERT INTO general_properties (
	// 		id, source, source_ad_id, created_at, updated_at, category, ad_link, company_ad, 
	// 		currency, images, list_time, body, subject, deal_type, remuneration_type, 
	// 		coordinates, city_or_district, region, price_byn, price_usd, price_eur, 
	// 		address, seller_name, contact_person, unp_number, company_address, company_license, import_link
	// 	) VALUES (
	// 		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, 
	// 		$16, $17, $18, $19, $20, $21, 
	// 		$22, $23, $24, $25, $26, $27, $28
	// 	)
	// 	ON CONFLICT (source, source_ad_id) DO UPDATE SET
	// 		updated_at = EXCLUDED.updated_at,
	// 		list_time = EXCLUDED.list_time,
	// 		price_byn = EXCLUDED.price_byn,
	// 		price_usd = EXCLUDED.price_usd,
	// 		price_eur = EXCLUDED.price_eur,
	// 		body = EXCLUDED.body,
	// 		subject = EXCLUDED.subject,
	// 		images = EXCLUDED.images
	// 	RETURNING id;
	// `
	// var propertyID uuid.UUID
	// err = tx.QueryRow(ctx, sqlGeneral,
	// 	dbGeneral.ID, dbGeneral.Source, dbGeneral.SourceAdID, dbGeneral.CreatedAt, dbGeneral.UpdatedAt, dbGeneral.Category, dbGeneral.AdLink, dbGeneral.CompanyAd,
	// 	dbGeneral.Currency, dbGeneral.Images, dbGeneral.ListTime, dbGeneral.Body, dbGeneral.Subject, dbGeneral.DealType, dbGeneral.RemunerationType,
	// 	dbGeneral.Coordinates, dbGeneral.CityOrDistrict, dbGeneral.Region, dbGeneral.PriceBYN, dbGeneral.PriceUSD, dbGeneral.PriceEUR,
	// 	dbGeneral.Address, dbGeneral.SellerName, dbGeneral.ContactPerson, dbGeneral.UnpNumber, dbGeneral.CompanyAddress, dbGeneral.CompanyLicense, dbGeneral.ImportLink,
	// ).Scan(&propertyID)

	// if err != nil {
	// 	return fmt.Errorf("failed to insert/update general_properties: %w", err)
	// }

	// //--- Шаг 2: Вставляем детали в зависимости от типа ---
	// if record.Details != nil {
	// 	switch details := record.Details.(type) {
	// 	case *domain.Apartment:
	// 		err = a.saveApartmentDetails(ctx, tx, propertyID, details)
	// 	case *domain.House:
	// 		err = a.saveHouseDetails(ctx, tx, propertyID, details)
	// 	case *domain.Commercial:
	// 		err = a.saveCommercialDetails(ctx, tx, propertyID, details)
	// 	case *domain.GarageAndParking:
	// 		err = a.saveGarageAndParkingDetails(ctx, tx, propertyID, details)
	// 	case *domain.Room:
	// 		err = a.saveRoomDetails(ctx, tx, propertyID, details)
	// 	case *domain.Plot:
	// 		err = a.savePlotDetails(ctx, tx, propertyID, details)
	// 	case *domain.NewBuilding:
	// 		err = a.saveNewBuildingDetails(ctx, tx, propertyID, details)
	// 	default:
	// 		log.Printf("Save warning: unknown details type for property %s. Skipping details insert.", propertyID)
	// 		return fmt.Errorf("save failed: unknown details type %T for source %s", record.Details, record.General.Source)
	// 	}
	// 	if err != nil {
	// 		return err 
	// 	}
	// }

	// // recordForSave, err := json.Marshal(dbGeneral)
	// // if err != nil {
	// // 	return err
	// // }

	// // Создаем папку, если ее нет
    //     // _ = os.MkdirAll("parsed_objects", 0755)
        
    //     // // Формируем имя файла на основе ad_id, чтобы избежать дубликатов
    //     // filename := fmt.Sprintf("parsed_objects/%d.json", record.General.SourceAdID)
        
    //     // // Записываем "сырое" тело ответа в файл
    //     // err = os.WriteFile(filename, recordForSave, 0644)
    //     // if err != nil {
    //     //     log.Printf("Failed to save response for ad_id %d: %v", record.General.SourceAdID, err)
    //     // } else {
    //     //     log.Printf("Successfully saved response for ad_id %d to %s", record.General.SourceAdID, filename)
    //     // }

	// 	// return nil
	return tx.Commit(ctx)
}


// --- Функции-мапперы: Domain -> DB Model ---

// func toDBGeneralProperty(d domain.GeneralProperty) domain.GeneralProperty {
	
// 	// wktPoint := fmt.Sprintf("SRID=4326;POINT(%f %f)", d.Longitude, d.Latitude)

// 	return domain.GeneralProperty{
// 		ID:           uuid.New(),
// 		Source:       d.Source,
// 		SourceAdID:   d.SourceAdID,
// 		CreatedAt:    time.Now(),
// 		UpdatedAt:    time.Now(),
// 		Category:     d.Category,
// 		AdLink:       d.AdLink,
// 		CompanyAd:    d.CompanyAd,
// 		Currency:     d.Currency,
// 		Images:       d.Images,
// 		ListTime:     d.ListTime,
// 		Body:         d.Body,
// 		Subject:      d.Subject,
// 		DealType:     d.DealType,
// 		RemunerationType: d.RemunerationType,
// 		Coordinates:  "",
// 		CityOrDistrict: d.CityOrDistrict,
// 		Region:       d.Region,
// 		PriceBYN:     d.PriceBYN,
// 		PriceUSD:     d.PriceUSD,
// 		PriceEUR:     d.PriceEUR,
// 		Address:      d.Address,
// 		SellerName:   d.SellerName,
// 		ContactPerson: d.ContactPerson,
// 		UnpNumber:    d.UnpNumber,
// 		CompanyAddress: d.CompanyAddress,
// 		CompanyLicense: d.CompanyLicense,
// 		ImportLink:   d.ImportLink,
// 	}
// }

// func toDBApartment(d *domain.Apartment) (domain.Apartment, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.Apartment{}, err
// 	}
// 	return domain.Apartment{
// 		RoomsAmount:           d.RoomsAmount,
// 		Condition:             d.Condition,
// 		BuildingFloors:        d.BuildingFloors,
// 		TotalArea:             d.TotalArea,
// 		YearBuilt:             d.YearBuilt,
// 		FloorNumber:           d.FloorNumber,
// 		PricePerSquareMeter:   d.PricePerSquareMeter,
// 		LivingSpaceArea:       d.LivingSpaceArea,
// 		KitchenSize:           d.KitchenSize,
// 		WallMaterial:          d.WallMaterial,
// 		Balcony:               d.Balcony,
// 		Bathroom:              d.Bathroom,
// 		FlatRepair:            d.FlatRepair,
// 		ContractNumberAndDate: d.ContractNumberAndDate,
// 		Parameters:            paramsJSON,
// 	}, nil
// }


// --- Функции для сохранения деталей ---

// func (a *PostgresStorageAdapter) saveApartmentDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.Apartment) error {
// 	// dbApt, err := toDBApartment(details)
// 	// if err != nil {
// 	// 	return fmt.Errorf("failed to marshal apartment parameters to json: %w", err)
// 	// }
// 	details.PropertyID = propID

// 	// Используем pgx.NamedArgs для удобства
// 	sql := `
// 		INSERT INTO apartments (
// 			property_id, rooms_amount, condition, building_floors, total_area, year_built, 
// 			floor_number, price_per_square_meter, living_space_area, kitchen_size, 
// 			wall_material, balcony, bathroom, flat_repair, contract_number_and_date, parameters
// 		) VALUES (
// 			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
// 		)
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			rooms_amount = EXCLUDED.rooms_amount,
// 			condition = EXCLUDED.condition,
// 			building_floors = EXCLUDED.building_floors,
// 			total_area = EXCLUDED.total_area,
// 			year_built = EXCLUDED.year_built,
// 			floor_number = EXCLUDED.floor_number,
// 			price_per_square_meter = EXCLUDED.price_per_square_meter,
// 			living_space_area = EXCLUDED.living_space_area,
// 			kitchen_size = EXCLUDED.kitchen_size,
// 			wall_material = EXCLUDED.wall_material,
// 			balcony = EXCLUDED.balcony,
// 			bathroom = EXCLUDED.bathroom,
// 			flat_repair = EXCLUDED.flat_repair,
// 			contract_number_and_date = EXCLUDED.contract_number_and_date,
// 			parameters = EXCLUDED.parameters;
// 	`
// 	_, err := tx.Exec(ctx, sql, details.PropertyID, details.RoomsAmount, details.Condition, details.BuildingFloors, details.TotalArea,
// 		details.YearBuilt, details.FloorNumber, details.PricePerSquareMeter, details.LivingSpaceArea, details.KitchenSize, details.WallMaterial,
// 		details.Balcony, details.Bathroom, details.FlatRepair, details.ContractNumberAndDate, details.Parameters)
// 	if err != nil {
// 		return fmt.Errorf("failed to insert/update apartment details: %w", err)
// 	}
// 	return nil
// }

// func toDBHouse(d *domain.House) (domain.House, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.House{}, err
// 	}
// 	return domain.House{
// 		TotalArea: d.TotalArea,             
// 		PlotArea: d.PlotArea,              
// 		WallMaterial: d.WallMaterial,      
// 		Condition: d.Condition,             
// 		YearBuilt: d.YearBuilt,             
// 		LivingSpaceArea: d.LivingSpaceArea,       
// 		BuildingFloors:	d.BuildingFloors,        
// 		RoomsAmount: d.RoomsAmount,           
// 		KitchenSize: d.KitchenSize,           
// 		Electricity: d.Electricity,                    
// 		InGardeningCommunity: d.InGardeningCommunity,  	         
// 		Water: d.Water,                    
// 		Heating: d.Heating,                    
// 		Sewage: d.Sewage,                   
// 		Gaz: d.Gaz,                   
// 		RoofMaterial: d.RoofMaterial,          
// 		ContractNumberAndDate: d.ContractNumberAndDate,  	     
// 		HouseType: d.HouseType,                  
// 		Parameters: paramsJSON,
// 	}, nil
// }


// --- Функции для сохранения деталей ---

// func (a *PostgresStorageAdapter) saveHouseDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.House) error {
// 	// dbHs, err := toDBHouse(details)
// 	// if err != nil {
// 	// 	return fmt.Errorf("failed to marshal house parameters to json: %w", err)
// 	// }
// 	details.PropertyID = propID

// 	// Используем pgx.NamedArgs для удобства
// 	sql := `
// 		INSERT INTO houses (
// 			property_id, total_area, plot_area, wall_material, condition, year_built, 
// 			living_space_area, building_floors, rooms_amount, kitchen_size, electricity,
// 			in_gardening_community, water, heating, sewage, gaz, roof_material, contract_number_and_date, 
// 			house_type, parameters
// 		) VALUES (
// 			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
// 		)
// 		ON CONFLICT (property_id) DO UPDATE SET
// 			total_area = EXCLUDED.total_area,
// 			plot_area = EXCLUDED.plot_area,
// 			wall_material = EXCLUDED.wall_material,
// 			condition = EXCLUDED.condition,
// 			year_built = EXCLUDED.year_built,
// 			living_space_area = EXCLUDED.living_space_area,
// 			building_floors = EXCLUDED.building_floors,
// 			rooms_amount = EXCLUDED.rooms_amount,
// 			kitchen_size = EXCLUDED.kitchen_size,
// 			electricity = EXCLUDED.electricity,
// 			in_gardening_community = EXCLUDED.in_gardening_community,
// 			water = EXCLUDED.water,
// 			heating = EXCLUDED.heating,
// 			sewage = EXCLUDED.sewage,
// 			gaz = EXCLUDED.gaz,
// 			roof_material = EXCLUDED.roof_material,
// 			contract_number_and_date = EXCLUDED.contract_number_and_date,
// 			house_type = EXCLUDED.house_type,
// 			parameters = EXCLUDED.parameters;
// 	`
// 	_, err := tx.Exec(ctx, sql, details.PropertyID, details.TotalArea, details.PlotArea, details.WallMaterial, details.Condition, details.YearBuilt,
// 		details.LivingSpaceArea, details.BuildingFloors, details.RoomsAmount, details.KitchenSize, details.Electricity, details.InGardeningCommunity,
// 		details.Water, details.Heating, details.Sewage, details.Gaz, details.RoofMaterial, details.ContractNumberAndDate, details.HouseType, details.Parameters)
// 	if err != nil {
// 		return fmt.Errorf("failed to insert/update house details: %w", err)
// 	}
// 	return nil
// }

// func toDBCommercial(d *domain.Commercial) (domain.Commercial, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.Commercial{}, err
// 	}
// 	return domain.Commercial{
// 		Condition: d.Condition,
// 		PropertyType: d.PropertyType,
// 		FloorNumber: d.FloorNumber,
// 		BuildingFloors: d.BuildingFloors,
// 		TotalArea: d.TotalArea,
// 		CommercialImprovements: d.CommercialImprovements,
// 		CommercialRepair: d.CommercialRepair,
// 		IsPartlySellOrRent: d.IsPartlySellOrRent,
// 		PricePerSquareMeter: d.PricePerSquareMeter,
// 		ContractNumberAndDate: d.ContractNumberAndDate,
// 		RoomsAmount: d.RoomsAmount,
// 		CommercialBuildingLocation: d.CommercialBuildingLocation,
// 		CommercialRentType: d.CommercialRentType,
// 		Parameters: paramsJSON,
// 	}, nil
// }


func (a *PostgresStorageAdapter) saveCommercialDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.Commercial) error {
	// dbComm, err := toDBCommercial(details)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal commercial parameters to json: %w", err)
	// }
	details.PropertyID = propID

	// Используем pgx.NamedArgs для удобства
	sql := `
		INSERT INTO commercial (
			property_id, property_type, condition, floor_number, building_floors, total_area, 
			commercial_improvements, commercial_repair, partly_sell_or_rent, price_per_square_meter, contract_number_and_date,
			rooms_amount, commercial_building_location, commercial_rent_type, parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
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
	`
	_, err := tx.Exec(ctx, sql, details.PropertyID, details.PropertyType, details.Condition, details.FloorNumber, details.BuildingFloors,
		details.TotalArea, details.CommercialImprovements, details.CommercialRepair, details.IsPartlySellOrRent, details.PricePerSquareMeter,
		details.ContractNumberAndDate, details.RoomsAmount, details.CommercialBuildingLocation, details.CommercialRentType, details.Parameters)
	if err != nil {
		return fmt.Errorf("failed to insert/update commercial details: %w", err)
	}
	return nil
}


// func toDBGarageAndParking(d *domain.GarageAndParking) (domain.GarageAndParking, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.GarageAndParking{}, err
// 	}
// 	return domain.GarageAndParking{
// 		PropertyType: d.PropertyType,
// 		ParkingPlacesAmount: d.ParkingPlacesAmount,
// 		TotalArea: d.TotalArea,
// 		Improvements: d.Improvements,
// 		Heating: d.Heating,
// 		ParkingType: d.ParkingType,
// 		Parameters: paramsJSON,
// 	}, nil
// }


func (a *PostgresStorageAdapter) saveGarageAndParkingDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.GarageAndParking) error {
	// dbGrg, err := toDBGarageAndParking(details)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal garage_and_parking parameters to json: %w", err)
	// }
	details.PropertyID = propID

	// Используем pgx.NamedArgs для удобства
	sql := `
		INSERT INTO garages_and_parkings (
			property_id, property_type, parking_places_amount, total_area, improvements, heating, parking_type, parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (property_id) DO UPDATE SET
			property_type = EXCLUDED.property_type,
			parking_places_amount = EXCLUDED.parking_places_amount,
			total_area = EXCLUDED.total_area,
			improvements = EXCLUDED.improvements,
			heating = EXCLUDED.heating,
			parking_type = EXCLUDED.parking_type,
			parameters = EXCLUDED.parameters;
	`
	_, err := tx.Exec(ctx, sql, details.PropertyID, details.PropertyType, details.ParkingPlacesAmount, details.TotalArea, details.Improvements,
		details.Heating, details.ParkingType, details.Parameters)
	if err != nil {
		return fmt.Errorf("failed to insert/update garage_and_parking details: %w", err)
	}
	return nil
}


// func toDBRoom(d *domain.Room) (domain.Room, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.Room{}, err
// 	}
// 	return domain.Room{
// 		Condition: d.Condition,                 
// 		Bathroom: d.Bathroom,                  
// 		SuggestedRoomsAmount: d.SuggestedRoomsAmount,      
// 		RoomsAmount: d.RoomsAmount,              
// 		FloorNumber: d.FloorNumber,               
// 		BuildingFloors: d.BuildingFloors,           
// 		TotalArea: d.TotalArea,                 
// 		IsBalcony: d.IsBalcony,                
// 		RentalType: d.RentalType,                
// 		LivingSpaceArea: d.LivingSpaceArea,           
// 		FlatRepair: d.FlatRepair,                
// 		IsFurniture: d.IsFurniture,               
// 		KitchenSize: d.KitchenSize,               
// 		KitchenItems: d.KitchenItems,              
// 		BathItems: d.BathItems,                 
// 		FlatRentForWhom: d.FlatRentForWhom,           
// 		FlatWindowsSide: d.FlatWindowsSide,           
// 		YearBuilt: d.YearBuilt,                
// 		WallMaterial: d.WallMaterial,              
// 		FlatImprovement: d.FlatImprovement,           
// 		RoomType: d.RoomType,                  
// 		ContractNumberAndDate: d.ContractNumberAndDate,     
// 		FlatBuildingImprovements: d.FlatBuildingImprovements,  
// 		Parameters: paramsJSON,
// 	}, nil
// }


func (a *PostgresStorageAdapter) saveRoomDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.Room) error {
	// dbRm, err := toDBRoom(details)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal room parameters to json: %w", err)
	// }
	details.PropertyID = propID

	// Используем pgx.NamedArgs для удобства
	sql := `
		INSERT INTO rooms (
			property_id, condition, bathroom, suggested_rooms_amount, rooms_amount, floor_number, building_floors, total_area, is_balcony,
			rental_type, living_space_area, flat_repair, is_furniture, kitchen_size, kitchen_items, bath_items, flat_rent_for_whom, 
			flat_windows_side, year_built, wall_material, flat_improvement, room_type, contract_number_and_date, flat_building_improvements,
			parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
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
	`
	_, err := tx.Exec(ctx, sql, details.PropertyID, details.Condition, details.Bathroom, details.SuggestedRoomsAmount, details.RoomsAmount, details.FloorNumber, 
		details.BuildingFloors, details.TotalArea, details.IsBalcony, details.RentalType, details.LivingSpaceArea, details.FlatRepair, details.IsFurniture,
		details.KitchenSize, details.KitchenItems, details.BathItems, details.FlatRentForWhom, details.FlatWindowsSide, details.YearBuilt, details.WallMaterial,
		details.FlatImprovement, details.RoomType, details.ContractNumberAndDate, details.FlatBuildingImprovements, details.Parameters)
	if err != nil {
		return fmt.Errorf("failed to insert/update room details: %w", err)
	}
	return nil
}

// func toDBPlot(d *domain.Plot) (domain.Plot, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.Plot{}, err
// 	}
// 	return domain.Plot{
// 		PlotArea: d.PlotArea,              
// 		InGardeningCommunity: d.InGardeningCommunity,    
// 		PropertyRights: d.PropertyRights,        
// 		Electricity: d.Electricity,           
// 		Water: d.Water,                      
// 		Gaz: d.Gaz,                         
// 		Sewage: d.Sewage,                   
// 		IsOutbuildings: d.IsOutbuildings,           
// 		OutbuildingsType: d.OutbuildingsType,            
// 		ContractNumberAndDate: d.ContractNumberAndDate,   	
// 		Parameters: paramsJSON,
// 	}, nil
// }


func (a *PostgresStorageAdapter) savePlotDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.Plot) error {
	// dbPlt, err := toDBPlot(details)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal plot parameters to json: %w", err)
	// }
	details.PropertyID = propID

	// Используем pgx.NamedArgs для удобства
	sql := `
		INSERT INTO plots (
			property_id, plot_area, in_gardening_community, property_rights, electricity, water, gaz, sewage, is_outbuildings,
			outbuildings_type, contract_number_and_date, parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
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
	`
	_, err := tx.Exec(ctx, sql, details.PropertyID, details.PlotArea, details.InGardeningCommunity, details.PropertyRights, details.Electricity, 
		details.Water, details.Gaz, details.Sewage, details.IsOutbuildings, details.OutbuildingsType, details.ContractNumberAndDate, details.Parameters)
	if err != nil {
		return fmt.Errorf("failed to insert/update plot details: %w", err)
	}
	return nil
}


// func toDBNewBuilding(d *domain.NewBuilding) (domain.NewBuilding, error) {
// 	paramsJSON, err := json.Marshal(d.Parameters)
// 	if err != nil {
// 		return domain.NewBuilding{}, err
// 	}
// 	return domain.NewBuilding{
// 		Deadline: d.Deadline,           
// 		RoomOptions: d.RoomOptions,         
// 		Builder: d.Builder,            
// 		ShareParticipation: d.ShareParticipation,  
// 		FloorOptions: d.FloorOptions,        
// 		WallMaterial: d.WallMaterial,       
// 		CeilingHeight: d.CeilingHeight,      
// 		LayoutOptions: d.LayoutOptions,       
// 		WithFinishing: d.WithFinishing,       
// 		Parameters: paramsJSON,
// 	}, nil
// }


func (a *PostgresStorageAdapter) saveNewBuildingDetails(ctx context.Context, tx pgx.Tx, propID uuid.UUID, details *domain.NewBuilding) error {
	// dbNewbld, err := toDBNewBuilding(details)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal new_building parameters to json: %w", err)
	// }
	details.PropertyID = propID

	// Используем pgx.NamedArgs для удобства
	sql := `
		INSERT INTO new_buildings (
			property_id, deadline, room_options, builder, share_participation, floor_options, wall_material, flat_ceiling_height,
			layout_options, with_finishing, parameters
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
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
	`
	_, err := tx.Exec(ctx, sql, details.PropertyID, details.Deadline, details.RoomOptions, details.Builder, details.ShareParticipation,
		details.FloorOptions, details.WallMaterial, details.CeilingHeight, details.LayoutOptions, details.WithFinishing, details.Parameters)
	if err != nil {
		return fmt.Errorf("failed to insert/update new_building details: %w", err)
	}
	return nil
}