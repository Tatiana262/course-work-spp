import type { IObjectCardResponse } from "./realEstateObjects";

export interface IFavoritesResponse {
    data: IObjectCardResponse[]; 
    total: number;               
    page: number;
    per_page: number;
}