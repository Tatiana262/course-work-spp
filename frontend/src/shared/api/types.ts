// Пример типа для пользователя, который мы будем хранить
export interface User {
    id: string;
    email: string;
    role: 'user' | 'admin';
}


export interface ObjectCard {
  id: string;
  master_object_id: string;
  title: string;
  priceUSD: number;
  images: string[];
  address: string;
  status: string;
}

export interface PaginatedObjectsResponse {
  data: ObjectCard[];
  total: number;
  page: number;
  perPage: number;
}

export interface Dictionaries {
    categories: { systemName: string; displayName: string }[];
    regions: { systemName: string; displayName: string }[];
    dealTypes: { systemName: string; displayName: string }[];
}