import type { IFilterState } from "../types/filter";

// Ключи, которые в URL должны обрабатываться как массивы (rooms=1&rooms=2)
export const ARRAY_KEYS: (keyof IFilterState)[] = [
    'rooms', 'wallMaterials', 'repairState', 'bathroomType', 'balconyType',
    'houseTypes', 'totalFloors', 'roofMaterials', 'waterConditions', 
    'heatingConditions', 'electricityConditions', 'sewageConditions', 'gazConditions',
    'commercialTypes', 'commercialBuildingLocations', 'commercialImprovements', 'commercialRepairs'
];

export const INITIAL_FILTERS: IFilterState = {
    category: 'apartment',
    dealType: 'sale',
    region: '',
    cityOrDistrict: '',
    street: '',
    priceCurrency: 'USD',
    priceMin: '', priceMax: '',
    rooms: [],

    totalAreaMin: '', totalAreaMax: '',
    livingSpaceAreaMin: '', livingSpaceAreaMax: '',
    kitchenAreaMin: '', kitchenAreaMax: '',
    yearBuiltMin: '', yearBuiltMax: '',
    wallMaterials: [],
    
    floorMin: '', floorMax: '',
    floorBuildingMin: '', floorBuildingMax: '',
    repairState: [],
    bathroomType: [],
    balconyType: [],

    houseTypes: [],
    plotAreaMin: '', plotAreaMax: '',
    totalFloors: [],
    roofMaterials: [],
    waterConditions: [],
    heatingConditions: [],
    electricityConditions: [],
    sewageConditions: [],
    gazConditions: [],

    commercialTypes: [],
    commercialImprovements: [],
    commercialRepairs: [],
    commercialBuildingLocations: [],
    roomsMin: '',
    roomsMax: '',
};