package kufarfetcher

import (
	"strings"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NormalizeStringPtr(s *string) *string {
    if s == nil {
        return nil
    }
    normalized := NormalizeFilterValue(*s)
    return &normalized
}

// NormalizeFilterValue очищает и стандартизирует строковое значение для фильтров
func NormalizeFilterValue(s string) string {
	if s == "" {
		return ""
	}

	lowerTrimmed := strings.ToLower(strings.TrimSpace(s))

	runes := []rune(lowerTrimmed)
	caser := cases.Upper(language.Russian) // Используем правила для русского/белорусского
	
	// Преобразуем только первую руну
	firstRuneTitle := []rune(caser.String(string(runes[0])))
	runes[0] = firstRuneTitle[0]
	
	return string(runes)
}

// NormalizeStringSlice применяет NormalizeFilterValue к каждому элементу среза
func NormalizeStringSlice(slice []string) []string {
    if slice == nil {
        return nil
    }
    normalized := make([]string, len(slice))
    for i, s := range slice {
        normalized[i] = NormalizeFilterValue(s)
    }
    return normalized
}



func NormalizeRegion(rawRegion string) string {
	// Убираем лишние пробелы в начале и в конце
	cleanRegion := strings.TrimSpace(rawRegion)
	
	// Убираем точки
	region := strings.ReplaceAll(cleanRegion, ".", "")

	// Заменяем сокращения
	if strings.HasSuffix(region, "обл") {
		// Обрезаем "обл" и добавляем " область"
		baseName := strings.TrimSpace(strings.TrimSuffix(region, "обл"))
		region = baseName + " область"
	}

	return region
}