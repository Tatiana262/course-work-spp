package postgres

import (
	"crypto/sha256"
	"fmt"
	// "regexp"
	// "sort"
	"storage-service/internal/core/domain"
	"strings"
	"github.com/mmcloughlin/geohash"
)

const geohashPrecision = 5 //?

// normalizeAddress упрощает адрес для стабильного хэширования
// var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9а-яА-Я]+`)

// func normalizeAddress(addr string) string {
// 	lower := strings.ToLower(addr)
// 	// Убираем слова-мусор
// 	replacer := strings.NewReplacer("улица", "", "ул", "", "дом", "", "д", "", "квартира", "", "кв", "")
// 	replaced := replacer.Replace(lower)
// 	// Оставляем только буквы и цифры
// 	return nonAlphanumericRegex.ReplaceAllString(replaced, "")
// }

func normalizeAreaToBucket(area *float64, bucketSize float64) string {
    if area == nil {
        return "null"
    }
    if bucketSize <= 0 {
        bucketSize = 1.0 // Защита от деления на ноль
    }
    bucketIndex := int(*area / bucketSize)
    return fmt.Sprintf("%d", bucketIndex)
}

// buildHashPayload создает стабильную строку из ключевых полей объекта для хэширования.
func buildHashPayload(rec domain.RealEstateRecord) string {

	geohsh := geohash.Encode(rec.General.Latitude, rec.General.Longitude)

	parts := []string{
		geohsh[:geohashPrecision],
		rec.General.Category,
		// rec.General.DealType,
		// normalizeAddress(rec.General.Address),
	}

	// Функция для безопасного добавления числовых указателей
	addInt := func(val *int8) {
		if val != nil {
			parts = append(parts, fmt.Sprintf("%d", *val))
		} else {
			parts = append(parts, "null")
		}
	}

	addFloat := func(val *float64) {
		if val != nil {
			parts = append(parts, fmt.Sprintf("%f", *val)) //убираю округление
		} else {
			parts = append(parts, "null")
		}
	}

	// Функция для безопасного добавления строковых указателей
	addString := func(val *string) {
		if val != nil && *val != "" {
			// Нормализуем строки: нижний регистр и убираем лишние пробелы
			parts = append(parts, strings.ToLower(strings.TrimSpace(*val)))
		} else {
			parts = append(parts, "null")
		}
	}

	addPart := func(part string) {
		parts = append(parts, part)
	}

	// Добавляем ключевые поля в зависимости от типа деталей
	switch d := rec.Details.(type) {
	// Квартира: Площадь, комнаты, этаж, этажность дома.
	case *domain.Apartment:
		addPart(normalizeAreaToBucket(d.TotalArea, 2.0))
		addInt(d.RoomsAmount)
		// addInt(d.FloorNumber)
		// addInt(d.BuildingFloors)
		fmt.Println(strings.Join(parts, "|"),  rec.General.ID)
		
	// Дом: Площадь дома, комнаты, площадь участка. Можно ещё AreaLiving
	case *domain.House:
		addPart(normalizeAreaToBucket(d.TotalArea, 2.0))
		addPart(normalizeAreaToBucket(d.PlotArea, 2.0))
		addInt(d.RoomsAmount)
		addString(d.HouseType)
		fmt.Println(strings.Join(parts, "|"), rec.General.ID)
		
		
	// Коммерческая недвижимость: Тип, площадь, этаж.
	case *domain.Commercial:
		addString(d.PropertyType)
		addFloat(d.TotalArea)
		// addInt(d.FloorNumber)
		// addInt(d.BuildingFloors)

	// Комната: Площадь самой комнаты, в квартире с каким кол-вом комнат она находится, и на каком этаже.
	case *domain.Room:
		addFloat(d.TotalArea) // Площадь самой комнаты
		// addInt(d.RoomsAmount) // Всего комнат в квартире
		// addInt(d.FloorNumber)
		// addInt(d.BuildingFloors)

	// Гараж/Парковка: Тип ("гараж" или "машиноместо") и площадь.
	case *domain.GarageAndParking:
		addString(d.PropertyType)
		addFloat(d.TotalArea)

	// Участок: Площадь участка - это его главная и самая стабильная характеристика.
	case *domain.Plot:
		addFloat(d.PlotArea)
		
	// Новостройка: Самый сложный случай. Объявления часто описывают не один объект, а весь проект.
	// Самыми стабильными идентификаторами проекта являются застройщик и, возможно, материал стен.
	// Адрес уже есть в базовых полях.
	case *domain.NewBuilding:
		addString(d.Builder)
	
	default:
		// Для типов без ключевых полей добавляем плейсхолдер
		// parts = append(parts, "no_details")
	}

	return strings.Join(parts, "|")
}

// calculateObjectHash вычисляет SHA256 хэш для объекта.
func calculateObjectHash(payload string) string {
	h := sha256.New()
	h.Write([]byte(payload))
	return fmt.Sprintf("%x", h.Sum(nil))
}