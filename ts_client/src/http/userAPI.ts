import { $authHost, $host } from "./index";
import { jwtDecode } from "jwt-decode";
import type { IUser } from "../store/UserStore"; // Импортируем интерфейс

// Описываем ответ от сервера при логине/регистрации
interface AuthResponse {
    token: string;
}

export const registration = async (email: string, password: string): Promise<IUser> => {
    // Указываем, что post возвращает AuthResponse
    const { data } = await $host.post<AuthResponse>('/auth/register', { email, password, role: 'user' });
    localStorage.setItem('token', data.token);
    return jwtDecode<IUser>(data.token); // Декодируем и говорим TS, что внутри IUser
}

export const login = async (email: string, password: string): Promise<IUser> => {
    const { data } = await $host.post<AuthResponse>('/auth/login', { email, password });
    localStorage.setItem('token', data.token);
    return jwtDecode<IUser>(data.token);
}

export const validate = async (): Promise<IUser> => {
    // Здесь бэкенд возвращает сразу JSON пользователя (IUser)
    const { data } = await $authHost.get<IUser>('/auth/validate');
    return data;
}