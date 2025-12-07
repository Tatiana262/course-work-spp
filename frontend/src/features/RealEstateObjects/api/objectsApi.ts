import { apiInstance } from '@/shared/api/axios';
import { PaginatedObjectsResponse, Dictionaries } from '@/shared/api/types';

// Функция для получения списка объектов
export const fetchObjects = async (filters: any): Promise<PaginatedObjectsResponse> => {
  const response = await apiInstance.get('/objects', { params: filters });
  return response.data;
};

// Функция для получения справочников
export const fetchDictionaries = async (): Promise<Dictionaries> => {
  const response = await apiInstance.get('/dictionaries');
  return response.data;
};