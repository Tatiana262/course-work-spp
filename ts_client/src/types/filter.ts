// Элемент справочника (как он приходит с бэка)
export interface IDictionaryItem {
    system_name: string; // или SystemName (зависит от JSON тегов в Go)
    display_name: string; // или DisplayName
}

// Ответ от сервера на запрос /dictionaries
export interface IDictionariesResponse {
    categories?: IDictionaryItem[];
    regions?: IDictionaryItem[];
    deal_types?: IDictionaryItem[];
    [key: string]: IDictionaryItem[] | undefined;
}

// То, что мы отправляем на бэкенд (Query Parameters)
export interface IFilterState {
    // Основные
    category: string;
    dealType: string;
    region: string;
    cityOrDistrict: string;
    street: string;
    priceCurrency: 'USD' | 'BYN' | 'EUR';
    priceMin: string; // Используем string для инпутов, при отправке преобразуем
    priceMax: string;
    rooms: number[]; // Массив выбранных комнат

    // Общие
    totalAreaMin: string;
    totalAreaMax: string;
    livingSpaceAreaMin: string;
    livingSpaceAreaMax: string;
    kitchenAreaMin: string;
    kitchenAreaMax: string;
    yearBuiltMin: string;
    yearBuiltMax: string;
    wallMaterials: string[];

    // Квартиры
    floorMin: string;
    floorMax: string;
    floorBuildingMin: string;
    floorBuildingMax: string;
    repairState: string[];
    bathroomType: string[];
    balconyType: string[];

    // Дома
    houseTypes: string[];
    plotAreaMin: string;
    plotAreaMax: string;
    totalFloors: string[]; // Если мультивыбор
    roofMaterials: string[];
    waterConditions: string[];
    heatingConditions: string[];
    electricityConditions: string[];
    sewageConditions: string[];
    gazConditions: string[];
}

// Опция фильтра, приходящая с бэкенда (из /filters/options)
export interface IFilterOptionData {
    min?: number;
    max?: number;
    options?: (string | number)[]; // Может быть массив строк или чисел
}

// Ответ от /filters/options
export interface IFilterOptionsResponse {
    filters: Record<string, IFilterOptionData>; // Ключи: "price", "cities", "rooms" и т.д.
    count: number;
}