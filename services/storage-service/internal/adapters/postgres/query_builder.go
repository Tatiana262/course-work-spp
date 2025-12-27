package postgres

import (
	"fmt"
	"strings"
	"storage-service/internal/core/domain"
)

type queryBuilder struct {
	joinClause strings.Builder
	conditions []string
	args       []interface{}
	argId      int
}

func newQueryBuilder() *queryBuilder {
	return &queryBuilder{
		argId:      1,
		conditions: []string{"gp.is_source_duplicate = false", "gp.status = 'active'"},
		args:       make([]interface{}, 0),
	}
}

func (qb *queryBuilder) addCondition(condition string, fieldName string, arg interface{}) {
	qb.conditions = append(qb.conditions, fmt.Sprintf(condition, fieldName, qb.argId))
	qb.args = append(qb.args, arg)
	qb.argId++
}

// AddFilter принимает указатели на float64 и int
func (qb *queryBuilder) AddFloatFilter(fieldName string, min *float64, max *float64) {
	if min != nil {
		qb.addCondition("%s >= $%d", fieldName, *min)
	}
	if max != nil {
		qb.addCondition("%s <= $%d", fieldName, *max)
	}
}

func (qb *queryBuilder) AddIntFilter(fieldName string, min *int, max *int) {
	if min != nil {
		qb.addCondition("%s >= $%d", fieldName, *min)
	}
	if max != nil {
		qb.addCondition("%s <= $%d", fieldName, *max)
	}
}

// build создает финальные части запроса
func (qb *queryBuilder) build() (string, string, []interface{}) {
	whereClause := ""
	if len(qb.conditions) > 0 {
		whereClause = "WHERE " + strings.Join(qb.conditions, " AND ")
	}
	return qb.joinClause.String(), whereClause, qb.args
}


// applyFilters - главный метод, который разбирает фильтры и строит запрос
func applyFilters(filters domain.FindObjectsFilters) (string, string, []interface{}) {
	qb := newQueryBuilder()

	
	// Фильтр по области (точное совпадение)
	if filters.Region != "" {
		qb.addCondition("%s = $%d", "gp.region", filters.Region)
	}

	// Фильтр по Городу/Району (точное совпадение)
	if filters.CityOrDistrict != "" {
		qb.addCondition("%s = $%d", "gp.city_or_district", filters.CityOrDistrict)
	}

	// Фильтр по улице (поиск подстроки в поле address)
	if filters.Street != "" {
		qb.addCondition("%s ILIKE $%d", "gp.address", "%"+filters.Street+"%")
	}

	// Основные фильтры по general_properties
	if filters.Category != "" {
		qb.addCondition("%s = $%d", "gp.category", filters.Category)
	}
	if filters.DealType != "" {
		qb.addCondition("%s = $%d", "gp.deal_type", filters.DealType)
	}

	switch filters.PriceCurrency {
	case "BYN":
		qb.AddFloatFilter("gp.price_byn", filters.PriceMin, filters.PriceMax)
	case "EUR":
		qb.AddFloatFilter("gp.price_eur", filters.PriceMin, filters.PriceMax)
	case "USD":
		qb.AddFloatFilter("gp.price_usd", filters.PriceMin, filters.PriceMax)
	}
	
	// Специфичные фильтры, требующие JOIN
	switch filters.Category {
	case "apartment":
		qb.joinClause.WriteString(" JOIN apartments d ON gp.id = d.property_id ")
		if len(filters.Rooms) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.rooms_amount", filters.Rooms)
		}

		qb.AddFloatFilter("d.total_area", filters.TotalAreaMin, filters.TotalAreaMax)

		qb.AddFloatFilter("d.living_space_area", filters.LivingSpaceAreaMin, filters.LivingSpaceAreaMax)
		qb.AddFloatFilter("d.kitchen_area", filters.KitchenAreaMin, filters.KitchenAreaMax)
		qb.AddIntFilter("d.year_built", filters.YearBuiltMin, filters.YearBuiltMax)
		
		if len(filters.WallMaterials) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.wall_material", filters.WallMaterials)
		}

		qb.AddIntFilter("d.floor_number", filters.FloorMin, filters.FloorMax)
		qb.AddIntFilter("d.building_floors", filters.FloorBuildingMin, filters.FloorBuildingMax)

		if len(filters.RepairState) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.repair_state", filters.RepairState)
		}

		if len(filters.BathroomType) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.bathroom_type", filters.BathroomType)
		}
		
		if len(filters.BalconyType) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.balcony_type", filters.BalconyType)
		}
	
	case "house":
		qb.joinClause.WriteString(" JOIN houses d ON gp.id = d.property_id ")
		if len(filters.Rooms) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.rooms_amount", filters.Rooms)
		}

		qb.AddFloatFilter("d.total_area", filters.TotalAreaMin, filters.TotalAreaMax)

		qb.AddFloatFilter("d.living_space_area", filters.LivingSpaceAreaMin, filters.LivingSpaceAreaMax)
		qb.AddFloatFilter("d.kitchen_area", filters.KitchenAreaMin, filters.KitchenAreaMax)
		qb.AddIntFilter("d.year_built", filters.YearBuiltMin, filters.YearBuiltMax)
		
		if len(filters.WallMaterials) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.wall_material", filters.WallMaterials)
		}

		if len(filters.HouseTypes) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.house_type", filters.HouseTypes)
		}

		qb.AddFloatFilter("d.plot_area", filters.PlotAreaMin, filters.PlotAreaMax)

		if len(filters.TotalFloors) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.building_floors", filters.TotalFloors)
		}
		if len(filters.RoofMaterials) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.roof_material", filters.RoofMaterials)
		}
		if len(filters.WaterConditions) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.water", filters.WaterConditions)
		}
		if len(filters.HeatingConditions) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.heating", filters.HeatingConditions)
		}
		if len(filters.ElectricityConditions) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.electricity", filters.ElectricityConditions)
		}
		if len(filters.SewageConditions) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.sewage", filters.SewageConditions)
		}
		if len(filters.GazConditions) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.gaz", filters.GazConditions)
		}
	case "commercial":
		qb.joinClause.WriteString(" JOIN commercial d ON gp.id = d.property_id ")

		if filters.PropertyType != "" {
			qb.addCondition("%s = $%d", "d.property_type", filters.PropertyType)
		}
		
		qb.AddIntFilter("d.floor_number", filters.FloorMin, filters.FloorMax)
		qb.AddIntFilter("d.building_floors", filters.FloorBuildingMin, filters.FloorBuildingMax)
		qb.AddFloatFilter("d.total_area", filters.TotalAreaMin, filters.TotalAreaMax)

		if len(filters.CommercialRepairs) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.commercial_repair", filters.CommercialRepairs)
		}

		if len(filters.CommercialLocation) > 0 {
			qb.addCondition("%s = ANY($%d)", "d.commercial_building_location", filters.CommercialLocation)
		}

		if len(filters.CommercialImprovements) > 0 {
			// Используем оператор пересечения &&
			qb.addCondition("%s && $%d", "d.commercial_improvements", filters.CommercialImprovements)
		}

		if filters.CommercialRoomsMin != nil || filters.CommercialRoomsMax != nil {
			// coalesce используется на случай, если одна из границ не задана
			condition := fmt.Sprintf(
				"(d.rooms_range[1] <= COALESCE($%d, 1000) AND d.rooms_range[cardinality(d.rooms_range)] >= COALESCE($%d, 0))",
				qb.argId, qb.argId+1,
			)
			qb.conditions = append(qb.conditions, condition)
			qb.args = append(qb.args, filters.CommercialRoomsMax, filters.CommercialRoomsMin)
			qb.argId += 2
		}
	}
	
	return qb.build()
}

