package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// ValueStat хранит статистику по одному значению поля.
type ValueStat struct {
	Value string
	Count int
}

// FieldStat хранит статистику по одному полю (ключу JSON).
type FieldStat struct {
	Name         string
	Count        int // Сколько раз поле было НЕ null
	ValueCounts  map[string]int
}

// CategoryStats хранит всю статистику для одной категории.
type CategoryStats struct {
	TotalFiles   int
	FieldStats   map[string]*FieldStat // Ключ - полное имя поля (e.g. "priceRates.840")
}

func main() {
	// --- НОВЫЙ БЛОК: ПЕРЕНАПРАВЛЕНИЕ ВЫВОДА В ФАЙЛ ---
	outputFile, err := os.Create("analysis_report_1.txt")
	if err != nil {
		log.Fatalf("Не удалось создать файл для отчета: %v", err)
	}
	defer outputFile.Close() // Гарантируем, что файл будет закрыт в конце

	// Перенаправляем стандартный вывод в наш файл
	log.SetOutput(outputFile)
	// Для fmt.Printf и fmt.Println мы будем использовать outputFile напрямую
	// --- КОНЕЦ НОВОГО БЛОКА ---


	searchPath := "api_responses_2/*/*.json"

	files, err := filepath.Glob(searchPath)
	if err != nil {
		log.Fatalf("Ошибка поиска файлов: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("Не найдено JSON файлов по пути: %s", searchPath)
	}
	fmt.Fprintf(outputFile, "Найдено %d файлов для анализа...\n", len(files))

	statsByCat := make(map[int]*CategoryStats)

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Ошибка чтения файла %s: %v", file, err)
			continue
		}

		var objectData map[string]interface{}
		if err := json.Unmarshal(data, &objectData); err != nil {
			log.Printf("Ошибка парсинга JSON в файле %s: %v", file, err)
			continue
		}

		categoryVal, ok := objectData["category"].(float64)
		if !ok {
			log.Printf("Не удалось определить категорию для файла %s. Пропускаем.", file)
			continue
		}
		categoryID := int(categoryVal)

		if _, exists := statsByCat[categoryID]; !exists {
			statsByCat[categoryID] = &CategoryStats{
				FieldStats: make(map[string]*FieldStat),
			}
		}

		stats := statsByCat[categoryID]
		stats.TotalFiles++
		collectStats(objectData, "", stats.FieldStats)
	}
	
	// Передаем outputFile в функцию печати
	printCategoryReports(statsByCat, outputFile)

	// Сообщение в консоль о завершении
	fmt.Println("Анализ завершен. Результаты сохранены в файл 'analysis_report.txt'")
}

// collectStats и valueToString остаются без изменений ...

// collectStats рекурсивно обходит JSON и собирает статистику.
func collectStats(node interface{}, prefix string, stats map[string]*FieldStat) {
	dataMap, ok := node.(map[string]interface{})
	if !ok {
		return
	}

	for key, value := range dataMap {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if key == "isNewBuild" && value != nil {

			if val, ok := value.(bool); ok && val {
				if code, ok := dataMap["code"].(float64); ok {
					cat := dataMap["category"].(float64)
					fmt.Printf("%d - %d\n", int64(cat), int64(code))
				} else {
					fmt.Println("get code error")
				}
			} 

			
	

			// if agency, ok := dataMap["agency"]; ok && agency == nil {
			// 	if seller, ok := dataMap["seller"].(string); ok && seller != "Агентство" {

			// 		if code, ok := dataMap["code"].(float64); ok {
			// 			cat := dataMap["category"].(float64)
			// 			fmt.Printf("%d - %d\n", int64(cat), int64(code))
			// 		} else {
			// 			fmt.Println("get code error")
			// 		}

			// 	}
				
			// }
		}

		if _, exists := stats[fullKey]; !exists {
			stats[fullKey] = &FieldStat{
				Name:        fullKey,
				ValueCounts: make(map[string]int),
			}
		}
		fieldStat := stats[fullKey]

		if value != nil {
			fieldStat.Count++
			
			valueStr := valueToString(value)
			fieldStat.ValueCounts[valueStr]++

			switch v := value.(type) {
			case map[string]interface{}:
				collectStats(v, fullKey, stats)
			case []interface{}:
				for _, item := range v {
					if subMap, ok := item.(map[string]interface{}); ok {
						collectStats(subMap, fullKey+"[]", stats)
					}
				}
			}
		}
	}
}

// valueToString преобразует любое значение в строку для статистики.
func valueToString(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		if len(val) > 70 {
			return `"` + val[:70] + `..."`
		}
		return `"` + val + `"`
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []interface{}:
		return fmt.Sprintf("[массив из %d элементов]", len(val))
	case map[string]interface{}:
		return "{объект}"
	default:
		return fmt.Sprintf("%T", v)
	}
}


// printCategoryReports теперь принимает *os.File для записи
func printCategoryReports(statsByCat map[int]*CategoryStats, w *os.File) {
	categoryNames := map[int]string{
		1: "Квартиры (аренда, сутки)", 2: "Квартиры (аренда, длительно)", 3: "Комнаты (аренда, сутки)", 4: "Комнаты (аренда, длительно)",
		5: "Квартиры (продажа)", 6: "Комнаты (продажа)", 7: "Дома (аренда, сутки)", 10: "Дома/Коттеджи (аренда, длительно)",
		11: "Дома/Коттеджи (продажа)", 12: "Новостройки (продажа)", 13: "Дачи (продажа)", 14: "Участки (продажа)",
	}

	sortedCatIDs := make([]int, 0, len(statsByCat))
	for catID := range statsByCat {
		sortedCatIDs = append(sortedCatIDs, catID)
	}
	sort.Ints(sortedCatIDs)

	fmt.Fprintln(w, "\n--- Отчет по анализу полей API Realt.by ---")

	for _, catID := range sortedCatIDs {
		stats := statsByCat[catID]
		catName := categoryNames[catID]

		fmt.Fprintf(w, "\n\n=================================================================\n")
		fmt.Fprintf(w, "  КАТЕГОРИЯ: %d (%s) | Проанализировано файлов: %d\n", catID, catName, stats.TotalFiles)
		fmt.Fprintf(w, "=================================================================\n")

		sortedFields := make([]*FieldStat, 0, len(stats.FieldStats))
		for _, fieldStat := range stats.FieldStats {
			sortedFields = append(sortedFields, fieldStat)
		}

		sort.Slice(sortedFields, func(i, j int) bool {
			if sortedFields[i].Count == sortedFields[j].Count {
				return sortedFields[i].Name < sortedFields[j].Name
			}
			return sortedFields[i].Count > sortedFields[j].Count
		})

		for _, field := range sortedFields {
			frequency := (float64(field.Count) / float64(stats.TotalFiles)) * 100.0
			fmt.Fprintf(w, "\n--- Поле: %-40s | Встречается в: %6.2f%% (%d/%d) ---\n", "'"+field.Name+"'", frequency, field.Count, stats.TotalFiles)
			
			sortedValues := make([]ValueStat, 0, len(field.ValueCounts))
			for val, count := range field.ValueCounts {
				sortedValues = append(sortedValues, ValueStat{Value: val, Count: count})
			}
			sort.Slice(sortedValues, func(i, j int) bool {
				return sortedValues[i].Count > sortedValues[j].Count
			})

			limit := 5
			for i, valStat := range sortedValues {
				if i >= limit {
					fmt.Fprintf(w, "    ... и еще %d других значений\n", len(sortedValues)-limit)
					break
				}
				fmt.Fprintf(w, "    - %-60s | %d раз\n", valStat.Value, valStat.Count)
			}
		}
	}
}