package constants

import (
	"fmt"
	"realt-parser-service/internal/core/domain"
)

// PredefinedSearch - это конкретная задача для парсинга.
type PredefinedSearch struct {
	Name     string
	Criteria domain.SearchCriteria
}


// PropertyType - наш внутренний "enum" для типов недвижимости.
type PropertyType int
const (
	
	PropertyTypeApartment = "flats"      // Квартира (вторичка)
	PropertyTypeNewBuilding = "zhiloy-kompleks"     // Квартира (новостройка)
	PropertyTypeCountryHouse = ""   // Дом, дача, коттедж
	PropertyTypePlot = ""           // Участок
	PropertyTypeRoom = ""           // Комната
)

// DealType - наш внутренний "enum" для типов сделок.
type DealType int
const (
	DealTypeSale = "sale"
	DealTypeRent = ""
)


const (
	ApartmentSaleCategory = 5
	ApartmentRentCategory = 2
	ApartmentRentForDayCategory = 1
	NewBuildingSaleCategory = 12

	CountryEstateSaleCategory = 11
	CountryEstateRentCategory = 7
	CountryEstateRentForDayCategory = 10
	DachaSaleCategory = 13
	PlotSaleCategory = 14

	RoomSaleCategory = 6
	RoomRentCategory = 4
	RoomRentForDayCategory = 3

	OfficeSaleCategory = 20
	OfficeRentCategory = 19
	WarehouseSaleCategory = 28
	WarehouseRentCategory = 22
	ShopSaleCategory = 18
	ShopRentCategory = 17
	RestaurantCafeSaleCategory = 32
	RestaurantCafeRentCategory = 26
	ServicesSaleCategory = 31
	ServicesRentCategory = 25
	BusinessSaleCategory = 30
	BusinessRentCategory = 24
	PomeschenieSaleCategory = 27
	PomeschenieRentCategory = 21
	ProizvodstvoSaleCategory = 29
	ProizvodstvoRentCategory = 23

	GarageSaleCategory = 16
	GarageRentCategory = 15
)



// searchConfigs - "словарь-переводчик" с языка домена на язык API Realt.by
var SearchConfigs = map[int]string{
	ApartmentSaleCategory: "sale-flats",
	ApartmentRentCategory: "rent-flat-for-long",
	ApartmentRentForDayCategory: "rent-flat-for-day",
	NewBuildingSaleCategory: "zhiloy-kompleks",

	CountryEstateSaleCategory: "sale-cottages",
	CountryEstateRentCategory: "rent-cottage-for-long",
	CountryEstateRentForDayCategory: "rent-cottage-for-day",
	DachaSaleCategory: "sale-dachi",
	PlotSaleCategory: "sale-plots",

	RoomSaleCategory:  "sale-rooms",
	RoomRentCategory: "rent-rooms-for-long", 
	RoomRentForDayCategory: "rent-rooms-for-day", 

	OfficeSaleCategory: "sale-offices",
	OfficeRentCategory: "rent-offices",

	WarehouseSaleCategory: "sale-warehouses",
	WarehouseRentCategory: "rent-warehouses",
	ShopSaleCategory: "sale-shops",
	ShopRentCategory: "rent-shops",
	RestaurantCafeSaleCategory: "sale-restorant-cafe",
	RestaurantCafeRentCategory: "rent-restorant-cafe",
	ServicesSaleCategory: "sale-services",
	ServicesRentCategory: "rent-services",
	BusinessSaleCategory: "sale-business",
	BusinessRentCategory: "rent-business",
	PomeschenieSaleCategory: "sale-pomeschenie",
	PomeschenieRentCategory: "rent-pomeschenie",
	ProizvodstvoSaleCategory: "sale-proizvodstvo",
	ProizvodstvoRentCategory: "rent-proizvodstvo",

	GarageSaleCategory: "sale-garage",
	GarageRentCategory: "rent-garage",
}

// Константы для UUID городов (их можно вынести и в отдельный файл constants)
const (
	CityMinskUUID    = "4cb07174-7b00-11eb-8943-0cc47adabd66"
	CityBrestUUID    = "4c8f8db2-7b00-11eb-8943-0cc47adabd66"
	CityVitebskUUID  = "4c9236d8-7b00-11eb-8943-0cc47adabd66"
	CityGomelUUID    = "4c95d414-7b00-11eb-8943-0cc47adabd66"
	CityGrodnoUUID   = "4c97eac6-7b00-11eb-8943-0cc47adabd66"
	CityMogilevUUID  = "4cb0e950-7b00-11eb-8943-0cc47adabd66"
)

const PageSize = 50
const SortBy = "updatedAt"
const SortOrder = "DESC"

// BusinessSearch описывает бизнес-задачу, которая может состоять из нескольких API-запросов.
type BusinessSearch struct {
	Name             string
	LocationUUID     string

	PropertyType string
	DealType     string

	Page         int    
	PageSize     int
	SortBy 		string
	SortOrder 	string
	// API-категории, которые соответствуют этой бизнес-задаче
	ApiCategories    []int
	// ApiObjectCategory нужен для новостроек
	ApiObjectCategory []int
	ApiObjectType	[]int

	Rooms []int
}


type SearchTaskTemplate struct {
	Name           string // Для логов, например "Продажа_ОфисныеЗдания"
	Category       int
	ObjectCategory []int
	ObjectType     []int
}

var RegionToRealtMap = map[string]string{
	"Минская область":       CityMinskUUID, 
	"Брестская область":       CityBrestUUID,     
	"Витебская область":     CityVitebskUUID,
	"Гомельская область":      CityGomelUUID,
	"Гродненская область":      CityGrodnoUUID,
	"Могилёвская область":     CityMogilevUUID,
}

var BusinessCategoryToTemplatesMap = map[string][]SearchTaskTemplate{
	"apartment": {
		{Name: "Продажа_Квартиры", Category: ApartmentSaleCategory},
		{Name: "Аренда_Квартиры", Category: ApartmentRentCategory},
		{Name: "АрендаНаСутки_Квартиры", Category: ApartmentRentForDayCategory},
	},
	"new_building": {
		{Name: "Продажа_Новостройки", Category: NewBuildingSaleCategory, ObjectCategory: []int{1}},
	},
	"house": {
		{Name: "Продажа_Коттеджи", Category: CountryEstateSaleCategory},
		{Name: "Продажа_Дачи", Category: DachaSaleCategory},
		{Name: "Аренда_Коттеджи", Category: CountryEstateRentCategory},
		{Name: "АрендаНаСутки_Коттеджи", Category: CountryEstateRentForDayCategory},
	},
    "plot": {
        {Name: "Продажа_Участки", Category: PlotSaleCategory},
    },
	"room": {
		{Name: "Продажа_Комнаты", Category: RoomSaleCategory},
		{Name: "Аренда_Комнаты", Category: RoomRentCategory},
		{Name: "АрендаНаСутки_Комнаты", Category: RoomRentForDayCategory},
	},
	"commercial": {
		// Каждая подкатегория коммерческой недвижимости - это отдельный шаблон
		{Name: "Продажа_Офисы", Category: OfficeSaleCategory},
		{Name: "Аренда_Офисы", Category: OfficeRentCategory},
		{Name: "Продажа_ОфисныеЗдания", Category: OfficeSaleCategory, ObjectType: []int{36, 21, 53, 40}},
		{Name: "Аренда_ОфисныеЗдания", Category: OfficeRentCategory, ObjectType: []int{36, 21, 53, 40}},
		{Name: "Продажа_СкладскиеЗдания", Category: WarehouseSaleCategory, ObjectType: []int{25, 26, 42, 43, 19, 48}},
        {Name: "Аренда_СкладскиеЗдания", Category: WarehouseRentCategory, ObjectType: []int{25, 26, 42, 43, 19, 48}},

		{Name: "Продажа_ТорговыеПомещения", Category: ShopSaleCategory, ObjectType: []int{27, 33, 15, 34, 22, 46, 47, 18, 31}},
        {Name: "Аренда_ТорговыеПомещения", Category: ShopRentCategory, ObjectType: []int{27, 33, 15, 34, 22, 46, 47, 18, 31}},

		{Name: "Продажа_РестораныКафе", Category: RestaurantCafeSaleCategory, ObjectType: []int{28, 38, 41}},
        {Name: "Аренда_РестораныКафе", Category: RestaurantCafeRentCategory, ObjectType: []int{28, 38, 41}},

		{Name: "Продажа_СфераУслуг", Category: ServicesSaleCategory, ObjectType: []int{30, 24, 29, 7, 39, 51, 49, 35, 9}},
        {Name: "Аренда_СфераУслуг", Category: ServicesRentCategory, ObjectType: []int{30, 24, 29, 7, 39, 51, 49, 35, 9}},

		{Name: "Продажа_Бизнес", Category: BusinessSaleCategory, ObjectType: []int{37, 20, 44, 32, 45}},
        {Name: "Аренда_Бизнес", Category: BusinessRentCategory, ObjectType: []int{37, 20, 44, 32, 45}},

		{Name: "Продажа_Здание", Category: PomeschenieSaleCategory, ObjectType: []int{16}},
        {Name: "Аренда_Здание", Category: PomeschenieRentCategory, ObjectType: []int{16}},

		{Name: "Продажа_Производство", Category: ProizvodstvoSaleCategory},
        {Name: "Аренда_Производство", Category: ProizvodstvoRentCategory},

	},
	"garage_parking": {
		{Name: "Продажа_ГаражМашиноместо", Category: GarageSaleCategory, ObjectType: []int{13, 14}},
		{Name: "Аренда_ГаражМашиноместо", Category: GarageRentCategory, ObjectType: []int{13, 14}},
	},
}

// GetBusinessSearches возвращает список бизнес-задач
func getBusinessSearches() []BusinessSearch {
	return []BusinessSearch{

		// {
		// 	Name:          "Продажа_Квартиры_Брест_5комнат",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ApartmentSaleCategory},
		// 	Rooms: []int{5},
		// },

		// {
		// 	Name:          "Аренда_Квартиры_Минск_5комнат",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ApartmentRentCategory, ApartmentRentForDayCategory},
		// 	Rooms: []int{5},
		// },



		// {
		// 	Name:              "Продажа_Новостройки_Минск",
		// 	LocationUUID:      CityMinskUUID,

		// 	PropertyType:  PropertyTypeNewBuilding,
		// 	DealType:      DealTypeSale,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,

		// 	ApiCategories:     []int{NewBuildingSaleCategory},
		// 	ApiObjectCategory: []int{1},
		// },
		
		// {
		// 	Name:          "Продажа_КоттеджДомПолдомаДачаТаунхаус_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{CountryEstateSaleCategory, DachaSaleCategory},
		// },
		// {
		// 	Name:          "Продажа_Участок_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{PlotSaleCategory},
		// },
		// {
		// 	Name:          "Аренда_КоттеджДомПолдомаДачаАгроусадьбаБаняГостиницаБазаОтдыхаЖильёДляСтроителей_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{CountryEstateRentCategory, CountryEstateRentForDayCategory},
		// },
		// {
		// 	Name:          "Продажа_Комнаты_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{RoomSaleCategory},
		// },
		// {
		// 	Name:          "Аренда_Комнаты_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{RoomRentCategory, RoomRentForDayCategory},
		// },

		// {
		// 	Name:          "Продажа_Офисы_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{OfficeSaleCategory},
		// },

		// {
		// 	Name:          "Продажа_ОфисныеЗдания_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{OfficeSaleCategory},
		// 	ApiObjectType: []int{36, 21, 53, 40},
		// },
		// {
		// 	Name:          "Аренда_ОфисныеЗдания_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{OfficeRentCategory},
		// 	ApiObjectType: []int{36, 21, 53, 40},
		// },

		// {
		// 	Name:          "Продажа_СкладскиеЗдания_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{WarehouseSaleCategory},
		// 	ApiObjectType: []int{25, 26, 42, 43, 19, 48},
		// },

		// {
		// 	Name:          "Аренда_СкладскиеЗдания_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{WarehouseRentCategory},
		// 	ApiObjectType: []int{25, 26, 42, 43, 19, 48},
		// },

		// {
		// 	Name:          "Продажа_ТорговыеПомещения_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ShopSaleCategory},
		// 	ApiObjectType: []int{27, 33, 15, 34, 22, 46, 47, 18, 31},
		// },

		// {
		// 	Name:          "Аренда_ТорговыеПомещения_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ShopRentCategory},
		// 	ApiObjectType: []int{27, 33, 15, 34, 22, 46, 47, 18, 31},
		// },

		// {
		// 	Name:          "Продажа_РестораныКафе_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{RestaurantCafeSaleCategory},
		// 	ApiObjectType: []int{28, 38, 41},
		// },

		// {
		// 	Name:          "Аренда_РестораныКафе_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{RestaurantCafeRentCategory},
		// 	ApiObjectType: []int{28, 38, 41},
		// },

		// {
		// 	Name:          "Продажа_СфераУслуг_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ServicesSaleCategory},
		// 	ApiObjectType: []int{30, 24, 29, 7, 39, 51, 49, 35, 9},
		// },

		// {
		// 	Name:          "Аренда_СфераУслуг_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ServicesRentCategory},
		// 	ApiObjectType: []int{30, 24, 29, 7, 39, 51, 49, 35, 9},
		// },

		// {
		// 	Name:          "Продажа_Бизнес_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{BusinessSaleCategory},
		// 	ApiObjectType: []int{37, 20, 44, 32, 45},
		// },

		// {
		// 	Name:          "Аренда_Бизнес_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{BusinessRentCategory},
		// 	ApiObjectType: []int{37, 20, 44, 32, 45},
		// },

		// {
		// 	Name:          "Продажа_ГаражМашиноместо_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{GarageSaleCategory},
		// 	ApiObjectType: []int{13, 14},
		// },

		// {
		// 	Name:          "Аренда_ГаражМашиноместо_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{GarageRentCategory},
		// 	ApiObjectType: []int{13, 14},
		// },

		// {
		// 	Name:          "Продажа_Здание_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{PomeschenieSaleCategory},
		// 	ApiObjectType: []int{16},
		// },

		// {
		// 	Name:          "Аренда_Здание_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{PomeschenieRentCategory},
		// 	ApiObjectType: []int{16},
		// },

		// {
		// 	Name:          "Продажа_Производство_Минск",
		// 	LocationUUID:  CityMinskUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ProizvodstvoSaleCategory},
		// },

		// {
		// 	Name:          "Аренда_Производство_Брест",
		// 	LocationUUID:  CityBrestUUID,
		// 	Page: 1,
		// 	PageSize: PageSize,
		// 	SortBy: SortBy,
		// 	SortOrder: SortOrder,
			
		// 	ApiCategories: []int{ProizvodstvoRentCategory},
		// },

	}
}





// GetSearchTasks - это ЕДИНСТВЕННАЯ функция, которую будет вызывать main.
// Она возвращает готовый список конкретных задач для адаптера.
func GetSearchTasks() []domain.SearchCriteria {
	businessSearches := getBusinessSearches()
	var tasks []domain.SearchCriteria

	for _, bs := range businessSearches {
		for _, category := range bs.ApiCategories {
			// Создаем уникальное имя для каждой подзадачи для логов
			// Создаем уникальное имя для каждой подзадачи для логов
			taskName := fmt.Sprintf("%s (cat: %d)", bs.Name, category)
			
			task := domain.SearchCriteria{
				Name:           taskName,
				LocationUUID:   bs.LocationUUID,
				Category:       category,
				ObjectCategory: bs.ApiObjectCategory,
				ObjectType:     bs.ApiObjectType,
				Page:           1, // Всегда начинаем с 1-й страницы

				Rooms: bs.Rooms,
			}
			tasks = append(tasks, task)
		}
	}
	return tasks
}