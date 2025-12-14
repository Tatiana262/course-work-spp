import { $authHost } from "./index";
import type { IFavoritesResponse } from "../types/favorites";

// Получить список избранного
export const fetchFavorites = async (page = 1, limit = 5): Promise<IFavoritesResponse> => {
    const offset = (page - 1) * limit;
    const { data } = await $authHost.get<IFavoritesResponse>('favorites', {
        params: { limit, offset }
    });
    return data;
};

// Получить только ID (для закрашивания сердечек)
export const fetchFavoriteIDs = async (): Promise<string[]> => {
    const { data } = await $authHost.get<string[]>('favorites/ids');
    return data; 
};

// Добавить в избранное
export const addToFavorites = async (masterObjectId: string) => {
    const { data } = await $authHost.post('favorites', {
        master_object_id: masterObjectId
    });
    return data;
};

// Удалить из избранного
export const removeFromFavorites = async (masterObjectId: string) => {
    const { data } = await $authHost.delete(`favorites/${masterObjectId}`);
    return data;
};
