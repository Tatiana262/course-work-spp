import { apiInstance } from '@/shared/api/axios';
import { User } from '@/shared/api/types';

// DTO для запросов. Хорошая практика - определять их явно.
export interface RegisterRequestDto {
  email: string;
  password: string;
}

export interface LoginRequestDto {
  email: string;
  password: string;
}

// DTO для ответов
interface AuthResponseDto {
  token: string;
  user_id: string;
  email: string;
  role: 'user' | 'admin';
}

// Функция для регистрации
export const registerUser = async (data: RegisterRequestDto): Promise<AuthResponseDto> => {
  const response = await apiInstance.post<AuthResponseDto>('/auth/register', data);
  return response.data;
};

// Функция для логина
export const loginUser = async (data: LoginRequestDto): Promise<AuthResponseDto> => {
  const response = await apiInstance.post<AuthResponseDto>('/auth/login', data);
  return response.data;
};