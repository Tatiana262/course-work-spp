package realtfetcher

import (
	"encoding/json"
	"fmt"
	// "log"
	"realt-parser-service/internal/constants"
	"realt-parser-service/internal/core/domain"
	"realt-parser-service/internal/core/port"

	// "strconv"
	"time"
)

// --- Структуры для парсинга __NEXT_DATA__ ---

type NextData struct {
	Props Props `json:"props"`
}

type Props struct {
	PageProps PageProps `json:"pageProps"`
}

type PageProps struct {
	InitialState InitialState `json:"initialState"`
}

type InitialState struct {
	ObjectView ObjectView `json:"objectView"`
}

type ObjectView struct {
	Object PropertiesObject `json:"object"`
}

type PriceHistoryItem struct {
	PriceCurrency int `json:"priceCurrency"`
}

type Agency struct {
	Title               string    `json:"title"`
	UNP                 int64     `json:"unp"`
	License             string    `json:"license"`
	LicensorDescription string    `json:"licensorDescription"`
	LicenseData         time.Time `json:"licenseData"`
}

type Agent struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type PropertiesObject struct {
	Code        int64  `json:"code"`
	Category    int    `json:"category"`
	TermsOfSale string `json:"termsOfSale"`

	PriceHistory []PriceHistoryItem `json:"normalizedPriceHistory"`
	Slides       []string           `json:"slides"` // Ссылки на фото
	CreatedAt    time.Time          `json:"createdAt"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Coordinates []float64 `json:"location"`

	StateRegionName   string `json:"stateRegionName"`
	StateDistrictName string `json:"stateDistrictName"`
	TownName          string `json:"townName"`

	PriceRates struct {
		USD float64 `json:"840"` // Цена в USD (код 840)
		BYN float64 `json:"933"` // Цена в BYN (код 933)
		RUB float64 `json:"643"` // Цена в RUB (код 643)
		EUR float64 `json:"978"` // Цена в RUB (код 978)
	} `json:"priceRates"`

	Address string `json:"address"`

	Agency *Agency `json:"agency"`
	Agent  *Agent  `json:"agent"`

	ContactName  *string `json:"contactName"`
	ContactEmail *string `json:"contactEmail"` // часто
	Seller       *string `json:"seller"`
	// если нету ContactName и Agency, взять значение из Seller
	ContactPhones []string `json:"contactPhones"` // всегда

	Apartment
	House
}

type AgencyConract struct {
	Contract string `json:"contract"`
}

type Apartment struct {
	RoomsAmount     *int8   `json:"rooms"`
	FloorNumber     *int8   `json:"storey"`
	BuildingFloorsAp  *int8   `json:"storeys"`
	TotalArea       *float64 `json:"areaTotal"`
	LivingSpaceArea *float64 `json:"areaLiving"`
	KitchenArea     *float64 `json:"areaKitchen"`
	YearBuilt       *int16   `json:"buildingYear"`
	WallMaterialAp  *string  `json:"houseType"`
	RepairState     *string  `json:"repairState"`
	BathroomType    *string  `json:"toilet"`
	BalconyType     *string  `json:"balconyType"`

	PriceRatesPerM2 struct {
		USD float64 `json:"840"` // Цена в USD (код 840)
		BYN float64 `json:"933"` // Цена в BYN (код 933)
		RUB float64 `json:"643"` // Цена в RUB (код 643)
		EUR float64 `json:"978"` // Цена в RUB (код 978)
	} `json:"priceRatesPerM2"`

	CeilingHeight        *float64       `json:"ceilingHeight"`
	Appliances           []string       `json:"appliances"`
	IsFencedTerritory    *bool          `json:"fencedTerritory"`
	HasFurniture         *bool          `json:"furniture"`
	HasGarage            *bool          `json:"garage"`
	IsAuction            *bool          `json:"isAuction"`
	IsNewBuild           *bool          `json:"isNewBuild"`
	// NearestMetroStations []string       `json:"nearestMetroStations"`
	HasParkingPlace      *bool          `json:"parkingPlace"`
	IsPriceHaggle        *bool          `json:"priceHaggle"`
	SeparateRooms        *int8          `json:"separateRooms"`
	HasSignaling         *bool          `json:"signaling"`
	StreetName           *string        `json:"streetName"`
	TownDistrictName     *string        `json:"townDistrictName"`
	TownSubDistrictName  *string        `json:"townSubDistrictName"`
	HasVideoIntercom     *bool          `json:"videoIntercom"`
	Views                []string       `json:"view"`
	AgencyConract        *AgencyConract `json:"agencyContract"`

	LeasePeriod			 *string        `json:"leasePeriod"`

	//TODO
}

type House struct {
	HasBath			 *bool		`json:"bath"`
	HasFireplace	 *bool		`json:"fireplace"`
	BuildingFloorsH  *int8   `json:"levels"`
	WallMaterialH *string  `json:"wallMaterial"`
	PlotArea     *float64 `json:"areaLand"`
	Electricity  *string  `json:"electricity"` // В Realt это строка ("есть")
	Water        *string  `json:"water"`
	Heating      *string  `json:"heating"`
	Sewage       *string  `json:"sewerage"`
	Gaz          *string  `json:"gas"`         // В Realt пишется "gas"
	RoofMaterial *string  `json:"roofMaterial"`
	HouseType    *string  `json:"objectType"` // "дом", "коттедж" и т.д.
	CompletionPercent  *int8   `json:"completionPercent"`
}

const (
	USD = 840
	BYN = 933
	EUR = 978
	RUB = 643
)

const (
	Sale = "sale"
	Rent = "rent"
)

// const (
// 	Apartments = "Квартиры"
// 	Houses = "Дома"
// )

var currenciesMap = map[int]string{
	USD: "USD",
	BYN: "BYN",
	EUR: "EUR",
	RUB: "RUB",
}

func toDomainRecord(jsonData string, url string, source string, logger port.LoggerPort) (*domain.RealEstateRecord, error) {
	var data NextData
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	obj := data.Props.PageProps.InitialState.ObjectView.Object

	// --- 1. Заполняем GeneralProperty ---
	general := domain.GeneralProperty{
		Source:           source,
		SourceAdID:       obj.Code,
		AdLink:           url,
		SaleType:        obj.TermsOfSale,

		ListTime:       obj.CreatedAt,
		Description:    obj.Description,
		Title:          obj.Title,
		Images:         obj.Slides,
		Longitude:      obj.Coordinates[0],
		Latitude:       obj.Coordinates[1],
		CityOrDistrict: obj.TownName,	//obj.StateDistrictName
		Region:         obj.StateRegionName,
		PriceBYN:       obj.PriceRates.BYN,
		PriceUSD:       obj.PriceRates.USD,
		PriceEUR:       &obj.PriceRates.EUR,

		Address:       obj.Address,
		SellerDetails: BuildSellerDetails(obj),

		Status:           domain.StatusActive,
	}

	switch obj.Category {
	case constants.ApartmentSaleCategory:
		// general.Category = Apartments
		general.DealType = Sale
	case constants.ApartmentRentCategory, constants.ApartmentRentForDayCategory:
		// general.Category = Apartments
		general.DealType = Rent

	case constants.CountryEstateSaleCategory, constants.DachaSaleCategory:
		// general.Category = Houses
		general.DealType = Sale
	case constants.CountryEstateRentCategory, constants.CountryEstateRentForDayCategory:
		// general.Category = Houses
		general.DealType = Rent
	}

	if len(obj.PriceHistory) != 0 {
		general.Currency = currenciesMap[obj.PriceHistory[0].PriceCurrency]
	}

	general.IsAgency = obj.Agency != nil
	if general.IsAgency {
		general.SellerName = obj.Agency.Title
	} else {
		if obj.ContactName != nil {
			general.SellerName = *obj.ContactName
		} else {
			general.SellerName = *obj.Seller
		}
	}

	var details interface{}

	switch obj.Category {
	case constants.ApartmentSaleCategory, constants.ApartmentRentCategory, constants.ApartmentRentForDayCategory:
		apt := &domain.Apartment{
			RoomsAmount:     obj.RoomsAmount,
			FloorNumber:     obj.FloorNumber,
			BuildingFloors:  obj.BuildingFloorsAp,
			TotalArea:       obj.TotalArea,
			LivingSpaceArea: obj.LivingSpaceArea,
			KitchenArea:     obj.KitchenArea,
			YearBuilt:       obj.YearBuilt,
			WallMaterial:    obj.WallMaterialAp,
			RepairState:     obj.RepairState,
			BathroomType:    obj.BathroomType,
			BalconyType:     obj.BalconyType,
			Parameters:      BuildApartmentParameters(obj),
		}

		if general.Currency == "BYN" {
			apt.PricePerSquareMeter = &obj.PriceRates.BYN
		} else {
			apt.PricePerSquareMeter = &obj.PriceRates.USD
		}

		details = apt

	case constants.CountryEstateSaleCategory, constants.DachaSaleCategory, 
		constants.CountryEstateRentCategory, constants.CountryEstateRentForDayCategory:   

	   house := &domain.House{
		   // Общие поля, которые есть и у квартир
		   TotalArea:       obj.TotalArea,
		   LivingSpaceArea: obj.LivingSpaceArea,
		   KitchenArea:     obj.KitchenArea,
		   YearBuilt:       obj.YearBuilt,
		   RoomsAmount:     obj.RoomsAmount,      
		   BuildingFloors:  obj.BuildingFloorsH,  
		   
		   // Специфичные для домов поля
		   PlotArea:     obj.PlotArea,
		   WallMaterial: obj.WallMaterialH, 
		   Electricity:  obj.Electricity,
		   Water:        obj.Water,
		   Heating:      obj.Heating,
		   Sewage:       obj.Sewage,
		   Gaz:          obj.Gaz,
		   RoofMaterial: obj.RoofMaterial,
		   HouseType:    obj.HouseType,
		   CompletionPercent: obj.CompletionPercent,
		   Parameters:   BuildHouseParameters(obj), // Новая функция для параметров дома
	   }
	   details = house

	default:
		logger.Warn("Unknown category received from Realt", port.Fields{
			"category_id": obj.Category,
			"ad_url":      url,
		})
	}

	record := &domain.RealEstateRecord{
		General: general,
		Details: details,
	}

	return record, nil
}

func BuildSellerDetails(g PropertiesObject) map[string]interface{} {
	seller := make(map[string]interface{})

	// Добавляем email, если есть
	if g.ContactEmail != nil && *g.ContactEmail != "" {
		seller["contactEmail"] = *g.ContactEmail
	}

	if len(g.ContactPhones) > 0 {
		seller["contactPhones"] = g.ContactPhones
	}

	// Добавляем агента, если есть
	if g.Agent != nil {
		seller["agent"] = g.Agent
	}

	if g.Agency != nil {
		seller["agency"] = g.Agency
	}

	return seller
}

func BuildApartmentParameters(obj PropertiesObject) map[string]interface{} {
	apartment := make(map[string]interface{})

	if obj.CeilingHeight != nil {
		apartment["ceiling_height"] = obj.CeilingHeight
	}

	if len(obj.Appliances) > 0 {
		apartment["appliances"] = obj.Appliances
	}

	if obj.IsFencedTerritory != nil {
		apartment["is_fenced_territory"] = obj.IsFencedTerritory
	}

	if obj.HasFurniture != nil {
		apartment["has_furniture"] = obj.HasFurniture
	}

	if obj.HasGarage != nil {
		apartment["has_garage"] = obj.HasGarage
	}

	if obj.IsAuction != nil {
		apartment["is_auction"] = obj.IsAuction
	}

	if obj.IsNewBuild != nil {
		apartment["is_new_build"] = obj.IsNewBuild
	}

	// if len(obj.NearestMetroStations) > 0 {
	// 	apartment["nearest_metro_stations"] = obj.NearestMetroStations
	// }

	if obj.HasParkingPlace != nil {
		apartment["has_parking_place"] = obj.HasParkingPlace
	}

	if obj.IsPriceHaggle != nil {
		apartment["is_price_haggle"] = obj.IsPriceHaggle
	}

	if obj.SeparateRooms != nil {
		apartment["separate_rooms"] = obj.SeparateRooms
	}

	if obj.HasSignaling != nil {
		apartment["has_signaling"] = obj.HasSignaling
	}

	if obj.StreetName != nil && *obj.StreetName != "" {
		apartment["street_name"] = *obj.StreetName
	}

	if obj.TownDistrictName != nil && *obj.TownDistrictName != "" {
		apartment["town_district_name"] = *obj.TownDistrictName
	}

	if obj.TownSubDistrictName != nil && *obj.TownSubDistrictName != "" {
		apartment["town_sub_district_name"] = *obj.TownSubDistrictName
	}

	if obj.HasVideoIntercom != nil {
		apartment["has_video_intercom"] = obj.HasVideoIntercom
	}

	if len(obj.Views) > 0 {
		apartment["views"] = obj.Views
	}

	if obj.AgencyConract != nil {
		apartment["agency_contract"] = obj.AgencyConract
	}

	if obj.LeasePeriod != nil {
		apartment["leasePeriod"] = obj.LeasePeriod
	}

	return apartment
}


func BuildHouseParameters(obj PropertiesObject) map[string]interface{} {
	params := make(map[string]interface{})

	// Используем уже существующие поля из Apartment struct, которые релевантны для дома
	if obj.HasFurniture != nil {
		params["has_furniture"] = *obj.HasFurniture
	}
	if obj.HasGarage != nil {
		params["has_garage"] = *obj.HasGarage
	}
	if obj.IsPriceHaggle != nil {
		params["is_price_haggle"] = *obj.IsPriceHaggle
	}
	if obj.LeasePeriod != nil {
		params["lease_period"] = *obj.LeasePeriod
	}
	if obj.AgencyConract != nil {
		params["agency_contract"] = obj.AgencyConract
	}
	if obj.HasFireplace != nil {
		params["has_fireplace"] = obj.HasFireplace
	}
	if obj.HasBath != nil {
		params["has_bath"] = obj.HasBath
	}
	
	return params
}

// // getInt16Ptr - хелпер для безопасного получения *int16 из карты.
// func getInt16Ptr(value interface{}) *int16 {
// 	// JSON числа всегда float64, их нужно конвертировать
// 	if val, ok := value.(float64); ok {
// 		v := int16(val)
// 		return &v
// 	}
// 	// Иногда Kufar отдает числа как строки
// 	if val, ok := value.(string); ok {
// 		if i, err := strconv.ParseInt(val, 10, 16); err == nil {
// 			v := int16(i)
// 			return &v
// 		}
// 	}
// 	// Иногда как массив
// 	if arVal, ok := value.([]interface{}); ok && len(arVal) == 1 {
// 		if val, ok1 := arVal[0].(float64); ok1 {
// 			v := int16(val)
// 			return &v
// 		}
// 	}

// 	return nil
// }

// func getFloat64Ptr(value interface{}) *float64 {
// 	if val, ok := value.(float64); ok {
// 		return &val
// 	}
// 	return nil
// }

// func getBoolPtr(value interface{}) *bool {
// 	if val, ok := value.(bool); ok {
// 		return &val
// 	}
// 	return nil
// }

// func getStringSlice(value interface{}) []string {
// 	// Сначала проверяем, что это срез интерфейсов
// 	if slice, ok := value.([]interface{}); ok {
// 		// Создаем новый срез строк нужной длины
// 		result := make([]string, 0, len(slice))
// 		// Проходим по каждому элементу
// 		for _, item := range slice {
// 			// Преобразуем каждый элемент в строку
// 			if str, ok := item.(string); ok {
// 				result = append(result, str)
// 			}
// 		}
// 		return result
// 	}
// 	// Если это не срез интерфейсов, возвращаем пустой срез
// 	return nil
// }

// func getInt16Slice(value interface{}) []int16 {
// 	// Проверяем, что это срез интерфейсов
// 	if slice, ok := value.([]interface{}); ok {
// 		// Создаем результат
// 		result := make([]int16, 0, len(slice))
// 		// Проходим по каждому элементу
// 		for _, item := range slice {
// 			// Пытаемся преобразовать элемент в float64
// 			if num, ok := item.(float64); ok {
// 				// Преобразуем float64 в int16
// 				result = append(result, int16(num))
// 			} else if str, ok := item.(string); ok {
//                 if i, err := strconv.ParseInt(str, 10, 16); err == nil {
//                     result = append(result, int16(i))
//                 }
//             }
// 		}
// 		return result
// 	}
// 	// Если это вообще не срез, возвращаем nil, чтобы можно было записать NULL в БД
// 	return nil
// }
