package kufarfetcher

import (
	"encoding/json"
	"fmt"

	"log"

	"kufar-parser-service/internal/core/domain"
	"strconv"
	"time"
)

//1010 - квартиры
//1020 - дома, коттеджи
//1030 - гаражи и стоянки
//1040 - комнаты
//1050 - коммерция
//1080 - участки (только продажа)
//1120 - новостройки

const (
	Sale = "sale"
	Rent = "rent"
)

// apiResponse - структура для разбора всего JSON ответа.
type apiResponse struct {
	Result struct {
		AccountParameters []apiParameter `json:"account_parameters"`
		AdParameters      []apiParameter `json:"ad_parameters"`

		// Основные поля
		AdID    int64  `json:"ad_id"`
		AdLink  string `json:"ad_link"`
		Body    string `json:"body"`
		Subject string `json:"subject"`

		CompanyAd bool       `json:"company_ad"`
		Currency  string     `json:"currency"`
		Images    []apiImage `json:"images"`
		ListTime  time.Time  `json:"list_time"`
		PriceBYN  string     `json:"price_byn"`
		PriceUSD  string     `json:"price_usd"`
		PriceEUR  string     `json:"price_eur"`
		Type      string     `json:"type"` // 'sell' или 'let'
	} `json:"result"`
}

// apiParameter - структура для одного элемента в массивах
type apiParameter struct {
	ParamName string `json:"p"`
	parameterValues
}

type parameterValues struct {
	ParamValue    interface{} `json:"v"`
	ParamAltValue interface{} `json:"vl"`
}

// apiImage - структура для одного изображения
type apiImage struct {
	Path string `json:"path"`
}


func toDomainRecord(jsonData []byte, source string) (*domain.RealEstateRecord, error) {
	var resp apiResponse
	if err := json.Unmarshal(jsonData, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal api response: %w", err)
	}

	adParams := paramsToMap(resp.Result.AdParameters)
	accountParams := paramsToMap(resp.Result.AccountParameters)

	general := domain.GeneralProperty{
		Source:     source,
		SourceAdID: resp.Result.AdID,
		AdLink:     resp.Result.AdLink,
		IsAgency:  resp.Result.CompanyAd,
		Currency:   resp.Result.Currency,
		ListTime:   resp.Result.ListTime,
		Body:       resp.Result.Body,
		Subject:    resp.Result.Subject,
		DealType:   resp.Result.Type,
		Images:     []string{},
	}

	if resp.Result.Type == "sell" {
		general.DealType = Sale
	} else {
		general.DealType = Rent
	}

	// Заполняем цены
	if price, err := parsePrice(resp.Result.PriceBYN); err == nil {
		general.PriceBYN = price
	} else {
		return nil, fmt.Errorf("could not parse required field PriceBYN: %w", err)
	}

	if price, err := parsePrice(resp.Result.PriceUSD); err == nil {
		general.PriceUSD = price
	} else {
		return nil, fmt.Errorf("could not parse required field PriceUSD: %w", err)
	}

	if price, err := parsePrice(resp.Result.PriceEUR); err == nil {
		general.PriceEUR = &price
	} else {
		return nil, fmt.Errorf("could not parse required field PriceEUR: %w", err)
	}

	for _, image := range resp.Result.Images {
		general.Images = append(general.Images, "https://rms5.kufar.by/v1/gallery/"+image.Path)
	}

	// Заполняем координаты
	if coords, ok := adParams["coordinates"].ParamValue.([]interface{}); ok && len(coords) == 2 {
		if lon, ok1 := coords[0].(float64); ok1 {
			if lat, ok2 := coords[1].(float64); ok2 {
				general.Longitude = lon
				general.Latitude = lat
			}
		}
	}

	general.RemunerationType, _ = adParams["remuneration_type"].ParamAltValue.(string)
	general.CityOrDistrict, _ = adParams["area"].ParamAltValue.(string)
	general.Region, _ = adParams["region"].ParamAltValue.(string)
	general.Category, _ = adParams["category"].ParamAltValue.(string)

	// Заполняем данные о продавце из accountParams
	general.SellerName, _ = accountParams["name"].ParamValue.(string)
	general.Address, _ = accountParams["address"].ParamValue.(string)
	general.SellerDetails = buildSellerDetails(accountParams)
	

	// Определяем категорию и создаем Details 
	var details interface{}
	category, _ := adParams["category"].ParamValue.(float64) // Категория приходит как число

	switch int(category) {
	case 1010: // Квартиры
		apt := &domain.Apartment{
			RoomsAmount:           getInt16Ptr(adParams["rooms"].ParamValue),
			// Condition:             getStringPtr(adParams["condition"].ParamAltValue),
			BuildingFloors:        getInt16Ptr(adParams["re_number_floors"].ParamValue),
			TotalArea:             getFloat64Ptr(adParams["size"].ParamValue),
			YearBuilt:             getInt16Ptr(adParams["year_built"].ParamValue),
			FloorNumber:           getInt16Ptr(adParams["floor"].ParamValue),
			PricePerSquareMeter:   getFloat64Ptr(adParams["square_meter"].ParamValue),
			LivingSpaceArea:       getFloat64Ptr(adParams["size_living_space"].ParamValue),
			KitchenArea:           getFloat64Ptr(adParams["size_kitchen"].ParamValue),
			WallMaterial:          getStringPtr(adParams["house_type"].ParamAltValue),
			Balcony:               getStringPtr(adParams["balcony"].ParamAltValue),
			BathroomType:              getStringPtr(adParams["bathroom"].ParamAltValue),
			RepairState:            getStringPtr(adParams["flat_repair"].ParamAltValue),
			// ContractNumberAndDate: getStringPtr(adParams["re_contract"].ParamValue),
		}
		// Собираем оставшиеся параметры в map
		apt.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
			"category", "rooms", "condition", "re_number_floors", "size", "year_built", "floor", "square_meter",
			"size_living_space", "size_kitchen", "house_type", "balcony", "bathroom", "flat_repair")
		details = apt

	case 1020: // Дома
		house := &domain.House{
			TotalArea:             getFloat64Ptr(adParams["size"].ParamValue),
			PlotArea:              getFloat64Ptr(adParams["size_area"].ParamValue),
			WallMaterial:          getStringPtr(adParams["wall_material"].ParamAltValue),
			Condition:             getStringPtr(adParams["condition"].ParamAltValue),
			YearBuilt:             getInt16Ptr(adParams["year_built"].ParamValue),
			LivingSpaceArea:       getFloat64Ptr(adParams["size_living_space"].ParamValue),
			BuildingFloors:        getInt16Ptr(adParams["house_number_floors"].ParamValue),
			KitchenSize:           getFloat64Ptr(adParams["size_kitchen"].ParamValue),
			Electricity:           getBoolPtr(adParams["electricity"].ParamValue),
			Water:                 getStringPtr(adParams["re_water"].ParamAltValue),
			Heating:               getStringPtr(adParams["re_heating"].ParamAltValue),
			Sewage:                getStringPtr(adParams["re_sewage"].ParamAltValue),
			Gaz:                   getStringPtr(adParams["house_gaz"].ParamAltValue),
			RoofMaterial:          getStringPtr(adParams["house_roof_material"].ParamAltValue),
			ContractNumberAndDate: getStringPtr(adParams["re_contract"].ParamValue),
		}

		if house.Gaz == nil {
			house.Gaz = getStringPtr(adParams["gaz"].ParamAltValue)
		}

		if general.DealType == "sell" {
			house.RoomsAmount = getInt16Ptr(adParams["rooms"].ParamValue)
		} else { //"let"
			house.RoomsAmount = getInt16Ptr(adParams["house_rent_rooms"].ParamValue)
		}

		if val := getInt16Ptr(adParams["re_garden_community"].ParamValue); val != nil {
			isInGardComm := *val == 2
			house.InGardeningCommunity = &isInGardComm
		}

		if general.DealType == "sell" {
			house.HouseType = getStringPtr(adParams["house_type_for_sell"].ParamAltValue)
		} else {
			house.HouseType = getStringPtr(adParams["house_type_for_rent"].ParamAltValue)
		}

		house.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
			"category", "size", "size_area", "wall_material", "condition", "year_built",
			"size_living_space", "house_number_floors", "size_kitchen", "electricity", "re_water", "re_heating", "re_sewage",
			"house_gaz", "house_roof_material", "re_contract", "gaz", "rooms", "house_rent_rooms", "re_garden_community",
			"house_type_for_sell", "house_type_for_rent")
		details = house

	case 1030: // гаражи и стоянки
		garage_or_parking := &domain.GarageAndParking{
			PropertyType:        getStringPtr(adParams["property_type"].ParamAltValue),
			ParkingPlacesAmount: getInt16Ptr(adParams["garage_parking_place"].ParamValue),
			TotalArea:           getFloat64Ptr(adParams["size"].ParamValue),
			Improvements:        getStringSlice(adParams["garage_improvements"].ParamAltValue),
			Heating:             getStringPtr(adParams["re_heating"].ParamAltValue),
			ParkingType:         getStringPtr(adParams["garage_parking_type"].ParamAltValue),
		}

		garage_or_parking.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
			"category", "house_gaz", "garage_parking_place", "size",
			"garage_improvements", "re_heating", "garage_parking_type")
		details = garage_or_parking

	case 1040:	//комнаты
		room := &domain.Room{
			RoomsAmount:              getInt16Ptr(adParams["rooms"].ParamValue),
			SuggestedRoomsAmount:     getInt16Ptr(adParams["rental_rooms"].ParamValue),
			Condition:                getStringPtr(adParams["condition"].ParamAltValue),
			Bathroom:                 getStringPtr(adParams["bathroom"].ParamAltValue),
			FloorNumber:              getInt16Ptr(adParams["floor"].ParamValue),
			TotalArea:                getFloat64Ptr(adParams["size"].ParamValue),
			BuildingFloors:           getInt16Ptr(adParams["re_number_floors"].ParamValue),
			RentalType:               getStringPtr(adParams["rental_type"].ParamAltValue),
			LivingSpaceArea:          getFloat64Ptr(adParams["size_living_space"].ParamValue),
			FlatRepair:               getStringPtr(adParams["flat_repair"].ParamAltValue),
			KitchenSize:              getFloat64Ptr(adParams["size_kitchen"].ParamValue),
			KitchenItems:             getStringSlice(adParams["flat_kitchen"].ParamAltValue),
			BathItems:                getStringSlice(adParams["flat_bath"].ParamAltValue),
			FlatRentForWhom:          getStringSlice(adParams["flat_rent_for_whom"].ParamAltValue),
			FlatWindowsSide:          getStringSlice(adParams["flat_windows_side"].ParamAltValue),
			YearBuilt:                getInt16Ptr(adParams["year_built"].ParamValue),
			WallMaterial:             getStringPtr(adParams["house_type"].ParamAltValue),
			FlatImprovement:          getStringSlice(adParams["flat_improvement"].ParamAltValue),
			RoomType:                 getStringPtr(adParams["room_type"].ParamAltValue),
			ContractNumberAndDate:    getStringPtr(adParams["re_contract"].ParamValue),
			FlatBuildingImprovements: getStringSlice(adParams["flat_building_improvements"].ParamAltValue),
		}

		if val := getInt16Ptr(adParams["is_balcony"].ParamValue); val != nil {
			isBalcony := *val == 1
			room.IsBalcony = &isBalcony
		}

		if val := getInt16Ptr(adParams["is_furniture"].ParamValue); val != nil {
			isFurniture := *val == 1
			room.IsFurniture = &isFurniture
		}

		room.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
			"category", "rooms", "rental_rooms", "condition", "bathroom", "floor", "size", "re_number_floors", "rental_type",
			"size_living_space", "flat_repair", "size_kitchen", "flat_kitchen", "flat_bath", "flat_rent_for_whom", "flat_windows_side",
			"year_built", "house_type", "flat_improvement", "room_type", "re_contract", "flat_building_improvements",
			"is_balcony", "is_furniture")
		details = room

	case 1050:	//коммерция
		commercial := &domain.Commercial{
			Condition:                getStringPtr(adParams["condition"].ParamAltValue),
			TotalArea:                getFloat64Ptr(adParams["size"].ParamValue),
			PropertyType:        	  getStringPtr(adParams["property_type"].ParamAltValue),
			PricePerSquareMeter:   	  getFloat64Ptr(adParams["square_meter"].ParamValue),
			FloorNumber:              getInt16Ptr(adParams["floor"].ParamValue),
			BuildingFloors:           getInt16Ptr(adParams["re_number_floors"].ParamValue),
			RoomsAmount:              getInt16Ptr(adParams["commercial_rooms"].ParamValue),
			ContractNumberAndDate:    getStringPtr(adParams["re_contract"].ParamValue),
			CommercialImprovements:   getStringSlice(adParams["commercial_improvements"].ParamAltValue),
			CommercialRepair:         getStringPtr(adParams["commercial_repair"].ParamAltValue),
			CommercialBuildingLocation: getStringPtr(adParams["commercial_building"].ParamAltValue),
			CommercialRentType:			getStringPtr(adParams["commercial_rent_type"].ParamAltValue),
		}

		if val := getInt16Ptr(adParams["commercial_partly_sell"].ParamValue); val != nil {
			isPartlySellOrRent := *val == 1
			commercial.IsPartlySellOrRent = &isPartlySellOrRent
		}

		commercial.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
		"category", "condition", "size", "property_type", "square_meter", "floor", "re_number_floors", "commercial_rooms",
		"re_contract", "commercial_improvements", "commercial_repair", "commercial_building", "commercial_rent_type",
		"commercial_partly_sell")
		details = commercial

	case 1080:	//участки
		plot := &domain.Plot{	
			PlotArea:              getFloat64Ptr(adParams["size_area"].ParamValue),
			PropertyRights:        getStringPtr(adParams["re_property_rights"].ParamAltValue),
			Electricity:           getStringPtr(adParams["re_electricity"].ParamAltValue),
			Water:                 getStringPtr(adParams["re_water"].ParamAltValue),
			Gaz:                   getStringPtr(adParams["re_gaz"].ParamAltValue),
			ContractNumberAndDate: getStringPtr(adParams["re_contract"].ParamValue),
			Sewage:				   getStringPtr(adParams["re_sewage"].ParamAltValue),
			IsOutbuildings: 	   getBoolPtr(adParams["re_outbuildings"].ParamValue),	
			OutbuildingsType: 	   getStringSlice(adParams["re_outbuildings_type"].ParamAltValue),
		}

		if val := getInt16Ptr(adParams["re_garden_community"].ParamValue); val != nil {
			isInGardenCommunity := *val == 2
			plot.InGardeningCommunity = &isInGardenCommunity
		}

		plot.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
		"category", "size_area", "re_property_rights", "re_electricity", "re_water", "re_gaz", "re_contract",
		"re_sewage", "re_outbuildings", "re_outbuildings_type")
		details = plot

	case 1120:	//новостройки
		newBuilding := &domain.NewBuilding{
			Deadline:           getStringPtr(adParams["new_buildings_year_built"].ParamAltValue),
			RoomOptions:        getInt16Slice(adParams["new_buildings_rooms"].ParamValue),
			Builder: 			getStringPtr(adParams["new_buildings_builder"].ParamAltValue),
			ShareParticipation: getBoolPtr(adParams["new_buildings_share_participation"].ParamValue),
			FloorOptions:       getInt16Slice(adParams["new_buildings_number_floors"].ParamValue),
			WallMaterial: 		getStringPtr(adParams["house_type"].ParamAltValue),
			CeilingHeight:      getStringPtr(adParams["flat_ceiling_height"].ParamAltValue),	
			LayoutOptions: 		getStringSlice(adParams["new_buildings_view"].ParamAltValue),	
			WithFinishing:      getBoolPtr(adParams["new_buildings_finishing"].ParamValue),
		}

		newBuilding.Parameters = getRemainingParams(adParams, "coordinates", "remuneration_type", "area", "region",
		"category", "new_buildings_year_built", "new_buildings_rooms", "new_buildings_builder", "new_buildings_share_participation",
		"new_buildings_number_floors", "house_type", "flat_ceiling_height", "new_buildings_view", "new_buildings_finishing")
		details = newBuilding

	default:
		log.Println("Unknown category")
	}

	// Собираем RealEstateRecord
	record := &domain.RealEstateRecord{
		General: general,
		Details: details,
	}

	return record, nil
}



func buildSellerDetails(accountParams map[string]parameterValues) map[string]interface{}{

	sellerDetails := make(map[string]interface{})
	
	contactPerson := getStringPtr(accountParams["contact_person"].ParamValue)
	if contactPerson != nil {
		sellerDetails["contact_person"] = contactPerson
	}

	companyAddress := getStringPtr(accountParams["company_address"].ParamValue)
	if companyAddress != nil {
		sellerDetails["company_address"] = companyAddress
	}

	unpNumber := getStringPtr(accountParams["vat_number"].ParamValue)
	if unpNumber != nil {
		sellerDetails["unp"] = unpNumber
	}

	companyLicense := getStringPtr(accountParams["company_number"].ParamValue)
	if companyLicense != nil {
		sellerDetails["company_license"] = companyLicense
	}

	importLink := getStringPtr(accountParams["import_link"].ParamValue)
	if importLink != nil {
		sellerDetails["import_link"] = importLink
	}
	
	return sellerDetails
}

// paramsToMap преобразует срез параметров в удобную карту
func paramsToMap(params []apiParameter) map[string]parameterValues {
	m := make(map[string]parameterValues)
	for _, p := range params {
		m[p.ParamName] = parameterValues{
			p.ParamValue,
			p.ParamAltValue,
		}
	}
	return m
}

// parsePrice преобразует цену из строки (в копейках) в float64
func parsePrice(s string) (float64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return float64(i) / 100.0, nil
}

// getStringPtr - хелпер для безопасного получения *string из карты
func getStringPtr(value interface{}) *string {
	if val, ok := value.(string); ok {
		return &val
	}
	return nil
}

// getInt16Ptr - хелпер для безопасного получения *int16 из карты
func getInt16Ptr(value interface{}) *int16 {
	// JSON числа всегда float64, их нужно конвертировать
	if val, ok := value.(float64); ok {
		v := int16(val)
		return &v
	}
	// Иногда Kufar отдает числа как строки
	if val, ok := value.(string); ok {
		if i, err := strconv.ParseInt(val, 10, 16); err == nil {
			v := int16(i)
			return &v
		}
	}
	// Иногда как массив
	if arVal, ok := value.([]interface{}); ok && len(arVal) == 1 {
		if val, ok1 := arVal[0].(float64); ok1 {
			v := int16(val)
			return &v
		}
	}

	return nil
}

func getFloat64Ptr(value interface{}) *float64 {
	if val, ok := value.(float64); ok {
		return &val
	}
	return nil
}

func getBoolPtr(value interface{}) *bool {
	if val, ok := value.(bool); ok {
		return &val
	}
	return nil
}

func getStringSlice(value interface{}) []string {
	
	if slice, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			// Преобразуем каждый элемент в строку
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	
	return nil
}

func getInt16Slice(value interface{}) []int16 {
	
	if slice, ok := value.([]interface{}); ok {
		result := make([]int16, 0, len(slice))
		for _, item := range slice {
			// Пытаемся преобразовать элемент в float64
			if num, ok := item.(float64); ok {
				// Преобразуем float64 в int16
				result = append(result, int16(num))
			} else if str, ok := item.(string); ok {
                if i, err := strconv.ParseInt(str, 10, 16); err == nil {
                    result = append(result, int16(i))
                }
            }
		}
		return result
	}

	return nil
}

// getRemainingParams принимает срез параметров и список ключей, которые нужно исключить, потому что мы их уже обработали
func getRemainingParams(params map[string]parameterValues, usedKeys ...string) map[string]interface{} {
	remaining := make(map[string]interface{})

	used := make(map[string]bool)
	for _, key := range usedKeys {
		used[key] = true
	}

	for key, values := range params {
		// Пропускаем уже обработанные ключи
		if _, ok := used[key]; ok {
			continue
		}

		switch key {
		case "flat_building_improvements", "flat_windows_side", "flat_kitchen", "flat_bath",
			"flat_improvement", "flat_rent_for_whom",
			"house_sell_area", "house_improvements", "house_rent_area", "house_rent_near_area", "house_rent_services":
			remaining[key] = getStringSlice(values.ParamAltValue)

		case "flat_ceiling_height", "flat_rent_prepayment", "new_buildings_apartment_complex",
			"re_property_rights", "house_roof_material_type", "house_readiness",
			"commercial_pavilions_type", "commercial_services_type",
			"re_special_purpose":
			remaining[key], _ = values.ParamAltValue.(string)

		case "trademark", "content_video", "re_contract":
			remaining[key], _ = values.ParamValue.(string)

		case "possible_exchange", "flat_new_building", "flat_open_room", "studio",
			"installment_pro", "re_auction_sale", "re_hot_water",
			"commercial_legal_address":
			remaining[key], _ = values.ParamValue.(bool)

		case "flat_furnished":
			val, _ := values.ParamValue.(bool)
			val_contr := !val
			remaining[key] = val_contr

		case "flat_rent_couchettes", "flat_storeys", "house_rent_couchettes":
			val, _ := values.ParamValue.(string)
			remaining[key], _ = strconv.ParseInt(val, 10, 64)

		case "size_snb", "re_outbuildings_size":
			remaining[key], _ = values.ParamValue.(float64)

		case "commercial_rent_workplace":
			val := getInt16Ptr(values.ParamValue)
			remaining[key] = *val == 1
		}
	}

	return remaining
}
