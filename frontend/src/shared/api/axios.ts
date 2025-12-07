import axios from 'axios';

// Получаем базовый URL для API Gateway из переменных окружения
const API_URL = import.meta.env.VITE_API_URL;

// Создаем инстанс axios с базовой конфигурацией
export const apiInstance = axios.create({
  baseURL: API_URL,
});

// === Interceptor для добавления Auth токена ===
// Этот "перехватчик" будет срабатывать перед КАЖДЫМ запросом
apiInstance.interceptors.request.use(
  (config) => {
    // Берем токен из localStorage (или другого хранилища)
    const token = localStorage.getItem('authToken');
    
    // Если токен есть, добавляем его в заголовок Authorization
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);