export interface IObjectCardResponse {
    id: string;
    title: string;
    price_usd: number;
    price_byn: number;
    address: string;
    images: string[]; // Предполагаем массив ссылок
    master_object_id: string;
    status: string;
    deal_type: string;
    category: string;
    // created_at: string;
    // добавь другие поля, которые возвращает бэкенд (площадь, этаж и т.д.)
}

export interface IPaginatedObjectsResponse {
    objects: IObjectCardResponse[];
    total: number;
    page: number;
    per_page: number;
}


export interface IObjectGeneralInfoResponse {
    master_object_id: string;
    id: string;
    source: string;
    source_ad_id: number;
    created_at: string;
    updated_at: string;
    category: string;
    ad_link: string;
    sale_type: string;
    currency: string;
    images: string[];
    list_time: string;
    description: string;
    title: string;
    deal_type: string;
    city_or_district: string;
    region: string;
    price_usd: number;
    price_byn: number;
    price_eur?: number;

    address: string;
    is_agency: boolean;
    seller_name: string;   

    status: string;

    seller_details: {
        // Поля могут быть разными, поэтому делаем их опциональными
        company_address?: string;
        contact_person?: string;
        unp?: string;
        contactPhones?: string[];
        contactEmail?: string;
        agency?: {
            title: string;
            license: string;
            unp: number;
        };
        [key: string]: any; 
    };
}

// Детали Квартиры (из domain.Apartment)
export interface IApartmentDetails {
    rooms_amount?: number;
    floor_number?: number;
    building_floors?: number;
    total_area?: number;
    living_space_area?: number;
    kitchen_area?: number;
    year_built?: number;
    wall_material?: string;
    repair_state?: string;
    bathroom_type?: string;
    balcony_type?: string;
    price_per_square_meter?: number;
    is_new_condition?: boolean;
    parameters: Record<string, any>; 
}

// Детали Дома (из domain.House)
export interface IHouseDetails {
    total_area?: number;
    plot_area?: number;
    wall_material?: string;
    year_built?: number;
    living_space_area?: number;
    building_floors?: number;
    rooms_amount?: number;
    kitchen_area?: number;
    electricity?: string;
    water?: string;
    heating?: string;
    sewage?: string;
    gaz?: string;
    roof_material?: string;
    house_type?: string;
    completion_percent?: string;
    is_new_condition?: boolean;
    parameters: Record<string, any>; 
}


export interface ICommercialDetails {
    property_type?: string;
    floor_number?: number;
    building_floors?: number;
    total_area?: number;
    commercial_improvements?: string[];
    commercial_repair?: string;
    price_per_square_meter?: number;
    rooms_range?: number[];
    commercial_building_location?: string;
    commercial_rent_type?: string;
    is_new_condition?: boolean;
    parameters: Record<string, any>; 
}

// Связанные предложения (DuplicatesInfoResponse)
export interface IRelatedOffer {
    id: string;
    source: string;
    ad_link: string;
    is_source_duplicate: boolean;
    deal_type: string;
}

// Итоговый ответ (ObjectDetailsResponse)
export interface IObjectDetailsResponse {
    general: IObjectGeneralInfoResponse;
    details: IApartmentDetails | IHouseDetails | ICommercialDetails; // TS сам разберется, что там
    related_offers: IRelatedOffer[]; // Проверь JSON тег в Go: RelatedOffers -> related_offers?
}