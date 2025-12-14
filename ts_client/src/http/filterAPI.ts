import { $host } from "./index";
import type { IDictionariesResponse, IFilterOptionsResponse, IFilterState } from "../types/filter";
import type { IObjectDetailsResponse } from "../types/realEstateObjects";

export const fetchDictionaries = async (): Promise<IDictionariesResponse> => {
    // Запрашиваем сразу 3 справочника
    const { data } = await $host.get<IDictionariesResponse>('/dictionaries', {
        params: {
            names: 'categories,regions,deal_types'
        }
    });
    return data;
}

export const fetchFilterOptions = async (filters: IFilterState): Promise<IFilterOptionsResponse> => {
    // Удаляем пустые ключи перед отправкой, чтобы URL был чище
    const params: any = {};
    Object.entries(filters).forEach(([key, value]) => {
        if (value !== "" && value !== null && (Array.isArray(value) ? value.length > 0 : true)) {
            params[key] = value;
        }
    });

    const { data } = await $host.get<IFilterOptionsResponse>('/filters/options', { params });
    // console.log(data)
    return data;
}