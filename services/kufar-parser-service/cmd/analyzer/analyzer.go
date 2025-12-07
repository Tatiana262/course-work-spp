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

// --- НОВЫЕ СТРУКТУРЫ ДЛЯ РАСШИРЕННОЙ СТАТИСТИКИ ---

// ValueStat хранит статистику по одному значению параметра.
type ValueStat struct {
	Value string
	Count int
}

// ParamStat хранит всю статистику по одному параметру (например, 'rooms').
type ParamStat struct {
	Name        string
	Count       int // Сколько раз параметр встретился
	ValueCounts map[string]int
}

// CategoryStats хранит всю статистику для одной категории.
type CategoryStats struct {
	TotalFiles int
	ParamStats map[string]*ParamStat // Ключ - имя параметра
}

func main() {
	// --- Перенаправление вывода в файл ---
	outputFile, err := os.Create("kufar_analysis_report.txt")
	if err != nil {
		log.Fatalf("Не удалось создать файл для отчета: %v", err)
	}
	defer outputFile.Close()
	log.SetOutput(outputFile)
	// ---

	// Карта для хранения статистики по каждой категории
	statsByCat := make(map[string]*CategoryStats)

	files, err := filepath.Glob("./api_responses/*.json")
	if err != nil {
		log.Fatalf("Failed to find json files: %v", err)
	}
	if len(files) == 0 {
		log.Fatal("No JSON files found in 'api_responses' directory.")
	}
	fmt.Fprintf(outputFile, "Найдено %d файлов для анализа...\n", len(files))

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		var root map[string]interface{}
		if err := json.Unmarshal(data, &root); err != nil {
			log.Printf("Error unmarshaling file %s: %v", file, err)
			continue
		}

		result, ok := root["result"].(map[string]interface{})
		if !ok {
			continue
		}

		var category string
		if params, ok := result["ad_parameters"].([]interface{}); ok {
			for _, p := range params {
				param, _ := p.(map[string]interface{})
				if pName, _ := param["p"].(string); pName == "category" {
					category = fmt.Sprintf("%.0f", param["v"]) // Категория - число
					break
				}
			}
		}

		if category == "" {
			log.Printf("Could not determine category for file %s. Skipping.", file)
			continue
		}

		if _, exists := statsByCat[category]; !exists {
			statsByCat[category] = &CategoryStats{
				ParamStats: make(map[string]*ParamStat),
			}
		}

		statsByCat[category].TotalFiles++
		if params, ok := result["ad_parameters"].([]interface{}); ok {
			processParametersForCategory(params, statsByCat[category].ParamStats)
		}
	}

	printCategoryReports(statsByCat, outputFile)
	fmt.Println("Анализ Kufar завершен. Результаты сохранены в файл 'kufar_analysis_report.txt'")
}

// processParametersForCategory теперь собирает и значения тоже.
func processParametersForCategory(params []interface{}, stats map[string]*ParamStat) {
	for _, p := range params {
		param, _ := p.(map[string]interface{})
		paramName, _ := param["p"].(string)

		// Инициализируем, если видим параметр впервые
		if _, exists := stats[paramName]; !exists {
			stats[paramName] = &ParamStat{
				Name:        paramName,
				ValueCounts: make(map[string]int),
			}
		}
		paramStat := stats[paramName]
		paramStat.Count++

		// В Kufar у параметра есть значение `v` и текстовое представление `vl`.
		// `vl` (value label) обычно более информативно.
		var valueToStore string
		if vl, ok := param["vl"]; ok && vl != nil {
			valueToStore = valueToString(vl)
		} else if v, ok := param["v"]; ok && v != nil {
			valueToStore = valueToString(v)
		} else {
			valueToStore = "NULL"
		}
		
		paramStat.ValueCounts[valueToStore]++
	}
}

// valueToString преобразует любое значение в строку для статистики.
func valueToString(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		if len(val) > 70 { return `"` + val[:70] + `..."` }
		return `"` + val + `"`
	case float64:
		if val == float64(int64(val)) { return strconv.FormatInt(int64(val), 10) }
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []interface{}:
		return fmt.Sprintf("[массив из %d элементов]", len(val))
	default:
		return fmt.Sprintf("%T", v)
	}
}

// printCategoryReports выводит отчеты с примерами значений.
func printCategoryReports(statsByCat map[string]*CategoryStats, w *os.File) {
	categoryNames := map[string]string{
		"1010": "Квартиры", "1020": "Дома, коттеджи", "1030": "Гаражи и стоянки",
		"1040": "Комнаты", "1050": "Коммерческая недвижимость",
		"1080": "Участки", "1120": "Новостройки",
	}
	
	sortedCats := make([]string, 0, len(statsByCat))
	for catName := range statsByCat {
		sortedCats = append(sortedCats, catName)
	}
	sort.Strings(sortedCats)

	fmt.Fprintln(w, "--- Отчет по анализу параметров API Kufar ---")

	for _, catName := range sortedCats {
		stats := statsByCat[catName]
		catTitle := categoryNames[catName]

		fmt.Fprintf(w, "\n\n=================================================================\n")
		fmt.Fprintf(w, "  КАТЕГОРИЯ: %s (%s) | Проанализировано файлов: %d\n", catName, catTitle, stats.TotalFiles)
		fmt.Fprintf(w, "=================================================================\n")

		sortedParams := make([]*ParamStat, 0, len(stats.ParamStats))
		for _, paramStat := range stats.ParamStats {
			sortedParams = append(sortedParams, paramStat)
		}
		sort.Slice(sortedParams, func(i, j int) bool {
			return sortedParams[i].Count > sortedParams[j].Count
		})

		for _, param := range sortedParams {
			frequency := (float64(param.Count) / float64(stats.TotalFiles)) * 100.0
			fmt.Fprintf(w, "\n--- Параметр: %-38s | Встречается в: %6.2f%% (%d/%d) ---\n", "'"+param.Name+"'", frequency, param.Count, stats.TotalFiles)
			
			sortedValues := make([]ValueStat, 0, len(param.ValueCounts))
			for val, count := range param.ValueCounts {
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

// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"path/filepath"
// 	"sort"
// )

// // Структура для хранения статистики по одному ключу/параметру
// type KeyStat struct {
// 	Name         string
// 	Count        int
// 	Frequency    float64
// 	ExampleValues map[interface{}]int // Карта для хранения примеров значений и их частоты
// }

// func main() {
// 	// Карты для подсчета частоты
// 	rootKeyFrequency := make(map[string]int)
// 	adParamFrequency := make(map[string]int)
// 	accountParamFrequency := make(map[string]int)

// 	// Карты для хранения примеров значений
// 	adParamExamples := make(map[string]map[interface{}]int)
// 	accountParamExamples := make(map[string]map[interface{}]int)

// 	totalFiles := 0

// 	files, err := filepath.Glob("./api_responses/*.json")
// 	if err != nil {
// 		log.Fatalf("Failed to find json files: %v", err)
// 	}
// 	if len(files) == 0 {
// 		log.Fatal("No JSON files found in 'api_responses' directory.")
// 	}


// 	for _, file := range files {
// 		totalFiles++
// 		data, err := ioutil.ReadFile(file)
// 		if err != nil {
// 			log.Printf("Error reading file %s: %v", file, err)
// 			continue
// 		}

// 		// Используем универсальный Unmarshal
// 		var root map[string]interface{}
// 		if err := json.Unmarshal(data, &root); err != nil {
// 			log.Printf("Error unmarshaling file %s: %v", file, err)
// 			continue
// 		}
		
// 		// Нас интересует только содержимое ключа "result"
// 		result, ok := root["result"].(map[string]interface{})
// 		if !ok {
// 			continue
// 		}


// 		// 1. Анализируем ключи верхнего уровня
// 		for key := range result {
// 			if key != "ad_parameters" && key != "account_parameters" {
// 				rootKeyFrequency[key]++
// 			}
// 		}

// 		// 2. Анализируем ad_parameters
// 		if params, ok := result["ad_parameters"].([]interface{}); ok {
// 			processParameters(params, adParamFrequency, adParamExamples)
// 		}

// 		// 3. Анализируем account_parameters
// 		if params, ok := result["account_parameters"].([]interface{}); ok {
// 			processParameters(params, accountParamFrequency, accountParamExamples)
// 		}
// 	}

// 	// Выводим результаты
// 	fmt.Printf("--- Analysis Report for %d JSON files ---\n\n", totalFiles)
// 	printSection("Core Fields (found in 'result')", rootKeyFrequency, nil, totalFiles)
// 	printSection("Ad Parameters (from 'ad_parameters' array)", adParamFrequency, adParamExamples, totalFiles)
// 	printSection("Account Parameters (from 'account_parameters' array)", accountParamFrequency, accountParamExamples, totalFiles)
// }

// // processParameters "распаковывает" массив параметров и собирает статистику
// func processParameters(params []interface{}, freq map[string]int, examples map[string]map[interface{}]int) {
// 	seenParams := make(map[string]bool) // Чтобы не считать один и тот же параметр дважды в одном файле
// 	for _, p := range params {
// 		param, ok := p.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}

// 		// Используем машинное имя параметра из поля "p"
// 		paramName, ok := param["p"].(string)
// 		if !ok {
// 			continue
// 		}

// 		if !seenParams[paramName] {
// 			freq[paramName]++
// 			seenParams[paramName] = true
// 		}

// 		// Собираем примеры значений из поля "v"
// 		if value, exists := param["v"]; exists {
// 			if examples[paramName] == nil {
// 				examples[paramName] = make(map[interface{}]int)
// 			}
// 			// Для простоты просто преобразуем значение в строку для хранения
// 			exampleKey := fmt.Sprintf("%v", value)
// 			examples[paramName][exampleKey]++
// 		}
// 	}
// }

// // printSection форматирует и выводит результаты для одной секции
// func printSection(title string, freq map[string]int, examples map[string]map[interface{}]int, total int) {
// 	fmt.Printf("--- %s ---\n", title)

// 	if len(freq) == 0 {
// 		fmt.Println("No data found for this section.")
// 		fmt.Println()
// 		return
// 	}

// 	stats := make([]KeyStat, 0, len(freq))
// 	for key, count := range freq {
// 		stat := KeyStat{
// 			Name:         key,
// 			Count:        count,
// 			Frequency:    (float64(count) / float64(total)) * 100.0,
// 		}
// 		if examples != nil {
// 			stat.ExampleValues = examples[key]
// 		}
// 		stats = append(stats, stat)
// 	}

// 	sort.Slice(stats, func(i, j int) bool {
// 		return stats[i].Count > stats[j].Count
// 	})

// 	for _, stat := range stats {
// 		fmt.Printf("Field: %-25s | Found in: %d files (%.2f%%)\n", "'"+stat.Name+"'", stat.Count, stat.Frequency)
// 		if stat.ExampleValues != nil {
// 			fmt.Printf("  Examples:\n")
// 			// Показываем не больше 3 примеров
// 			count := 0
// 			for val, num := range stat.ExampleValues {
// 				if count >= 3 {
// 					break
// 				}
// 				fmt.Printf("    - [%s] (seen %d times)\n", val, num)
// 				count++
// 			}
// 		}
// 	}
// 	fmt.Println()
// }