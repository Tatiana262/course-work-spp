package postgres

// import (
// 	"fmt"
// 	"math"
// 	"sort"
// 	"storage-service/internal/core/domain"
// 	"strings"

// 	// "github.com/mmcloughlin/geohash"
// )

// // Округляем до ближайшего целого
// func normalizeArea(a *float64) string {
//     if a == nil {
//         return "null"
//     }
//     // Используем math.Round для правильного округления
//     return fmt.Sprintf("%d", int(math.Round(*a))) 
// }

// // normalizeInt приводит *int16 к строке или "null"
// func normalizeInt(val *int16) string {
// 	if val == nil {
// 		return "null"
// 	}
// 	return fmt.Sprintf("%d", *val)
// }

// // buildCanonicalHash создает хэш из НЕ-географических данных
// func buildCanonicalHash(rec domain.RealEstateRecord) string {
//     // Эта функция должна быть похожа на ваш старый buildHashPayload,
//     // НО БЕЗ rec.General.Address и БЕЗ координат.
//     // Она должна включать только площадь, комнаты, этаж и т.д.
//     parts := []string{}
    
//     switch d := rec.Details.(type) {
// 	case *domain.Apartment:
// 		parts = append(parts, "type:apt")
// 		// parts = append(parts, "floor:"+normalizeInt(d.FloorNumber))
// 		parts = append(parts, "area:"+normalizeArea(d.TotalArea))
// 		parts = append(parts, "rooms:"+normalizeInt(d.RoomsAmount))
//     // ... и так далее для других типов
//     }

//     sort.Strings(parts) // Сортировка важна для стабильности
// 	payload := strings.Join(parts, "|")
//     return calculateObjectHash(payload)
// }

// // getCoordinatesFromRecord - вспомогательная функция для извлечения координат
// // Вам нужно будет реализовать ее в зависимости от того, как вы передаете координаты
// func getCoordinatesFromRecord(rec domain.RealEstateRecord) (lat, lon float64, err error) {

// 	if rec.General.Latitude == 0 && rec.General.Longitude == 0 {
// 		return 0, 0, fmt.Errorf("coordinates are zero in the record")
// 	}
// 	return rec.General.Latitude, rec.General.Longitude, nil
// }