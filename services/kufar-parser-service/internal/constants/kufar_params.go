package constants

import "kufar-parser-service/internal/core/domain"

// Categories
const (
    ApartmentCategory = "1010"
    HouseCategory = "1020"
    GarageAndParkingCategory = "1030"
    RoomCategory = "1040"
    CommercialCategory = "1050"
    PlotCategory = "1080"
    NewBuildingCategory = "1120"

    TravelsCategory = "25010"
    // TourismCategory = "13140"
)

// Deal Types
const (
	DealTypeSell = "sell"
	DealTypeRent = "let"
)

const MaxAdsAmount = 200

// Sort Options
const (
	SortByDateDesc = "lst.d" // List time descending
)

// Locations
const (
    Minsk = "country-belarus~province-minsk~locality-minsk"
    MinskRegion = "country-belarus~province-minskaja_oblast"
    BrestRegion = "country-belarus~province-brestskaja_oblast"
    // BrestRegion = "country-belarus~province-brestskaja_oblast~locality-brest" //для домов
    VitebskRegion = "country-belarus~province-vitebskaja_oblast"
    GomelRegion = "country-belarus~province-gomelskaja_oblast"
    GrodnoRegion = "country-belarus~province-grodnenskaja_oblast"
    MogilevRegion = "country-belarus~province-mogilyovskaja_oblast"
    Belarus = "country-belarus"
)


// Вы можете также создать структуры для более сложных наборов критериев,
// если планируете запускать парсер для множества предопределенных фильтров.
type PredefinedSearch struct {
    Name         string
    Criteria     domain.SearchCriteria // Используем вашу доменную структуру
}

// BusinessCategoryToKufarMap - наш главный "словарь-переводчик" для Kufar.
// Ключ - бизнес-категория, значение - технический ID категории на Kufar.
var BusinessCategoryToKufarMap = map[string][]string{
	"apartment":    {ApartmentCategory, TravelsCategory},
	"house":        {HouseCategory, TravelsCategory},
	"garage_parking": {GarageAndParkingCategory},
	"room":         {RoomCategory},
	"commercial":   {CommercialCategory},
	"plot":     {PlotCategory},
	"new_building": {NewBuildingCategory},
}

// RegionToKufarMap - теперь сопоставляет бизнес-регион со СРЕЗОМ технических локаций Kufar
var RegionToKufarMap = map[string][]string{
	"Минская область":       {Minsk, MinskRegion}, 
	"Брестская область":       {BrestRegion},     
	"Витебская область":     {VitebskRegion},
	"Гомельская область":      {GomelRegion},
	"Гродненская область":      {GrodnoRegion},
	"Могилёвская область":     {MogilevRegion},
	"Вся Беларусь": {Belarus},
}

var Queries = map[string]string{
    "apartment": "v.or:5",
    "house": "v.or:10,15",
}

var DealTypes = []string{DealTypeSell, DealTypeRent}

// GetPredefinedSearches возвращает список предопределенных наборов критериев для парсинга
func GetPredefinedSearches() []PredefinedSearch {
    return []PredefinedSearch{
        // {
        //     Name: "Квартиры_БрестскаяОбласть_Продажа",
        //     Criteria: domain.SearchCriteria{
        //         Category: ApartmentCategory,
        //         DealType: DealTypeSell,
        //         Location: BrestRegion,
        //         AdsAmount: MaxAdsAmount,            
        //         SortBy:   SortByDateDesc,         
        //     },
        // },
        // {
        //     Name: "Тест_Квартиры_Могилёв_5комнат",
        //     Criteria: domain.SearchCriteria{
        //         Category: ApartmentCategory,
        //         DealType: DealTypeSell,
        //         Location: "country-belarus~province-mogilyovskaja_oblast~locality-mogilyov",
        //         AdsAmount: MaxAdsAmount,            
        //         SortBy:   SortByDateDesc,         
        //     },
        // },
        // {
        //     Name: "Дома_МинскаяОбласть_Аренда",
        //     Criteria: domain.Criteria{
        //         Region:       RegionMinskayaOblast,
        //         DealType:     DealTypeRent,
        //         PropertyType: PropertyTypeHouse,
        //         SortBy:       SortByDateDesc,
        //     },
        // },
    }
}