import axios from "axios";
import type { InternalAxiosRequestConfig } from "axios";

// Используем import.meta.env для Vite или process.env для Webpack
const API_URL = import.meta.env.VITE_API_URL || "http://localhost:5000/"; 

const $host = axios.create({
    baseURL: API_URL,
    paramsSerializer: {
        indexes: null // null означает "не добавлять скобки [] к ключам массивов"
    }
});

const $authHost = axios.create({
    baseURL: API_URL,
});

const authInterceptor = (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
    const token = localStorage.getItem('token');
    if (token && config.headers) {
        config.headers.authorization = `Bearer ${token}`;
    }
    return config;
}

$authHost.interceptors.request.use(authInterceptor);

export {
    $host,
    $authHost
}