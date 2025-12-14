import { $host } from "./index";
import type { IObjectDetailsResponse, IPaginatedObjectsResponse } from "../types/realEstateObjects";

// Принимаем параметры фильтрации как объект
export const fetchObjects = async (params: any): Promise<IPaginatedObjectsResponse> => {
    const { data } = await $host.get<IPaginatedObjectsResponse>('/objects', {
        params: params // Axios сам превратит объект {category: '...'} в строку запроса
    });
    return data;
}


export const fetchObjectWithDeatils = async (id: string): Promise<IObjectDetailsResponse> => {
    // Твой эндпоинт: /api/v1/objects/{objectID}
    const { data } = await $host.get<IObjectDetailsResponse>(`/objects/${id}`);
    return data;
}