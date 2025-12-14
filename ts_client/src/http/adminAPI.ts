import { $authHost } from "./index";
import type { ITask, ITasksResponse } from "../types/task";
import { fetchEventSource } from '@microsoft/fetch-event-source';

// --- ЗАПУСК ЗАДАЧ (Actualization Service) ---

export const findNewObjects = async (categories: string[], regions: string[]) => {
    // POST /api/v1/actualize/new-objects
    const { data } = await $authHost.post('actualize/new-objects', { categories, regions });
    return data;
};

export const actualizeActive = async (category = "Квартиры", limit = 100) => {
    // POST /api/v1/actualize/active
    const { data } = await $authHost.post('actualize/active', { category, limit });
    return data;
};

// Для пользователя (актуализация одного объекта)
export const actualizeObject = async (masterObjectId: string) => {
    // POST /api/v1/actualize/object
    const { data } = await $authHost.post('actualize/object', { master_object_id: masterObjectId });
    return data;
};

// --- МОНИТОРИНГ ЗАДАЧ (Task Service) ---

export const fetchTasks = async (page = 1, perPage = 20): Promise<ITasksResponse> => {
    const { data } = await $authHost.get<ITasksResponse>('tasks', {
        params: { page, perPage }
    });
    return data;
};

export const fetchTaskById = async (id: string): Promise<ITask> => {
    const { data } = await $authHost.get<ITask>(`tasks/${id}`);
    return data;
};

// Подписка на SSE
export const subscribeToTasks = (
    onMessage: (task: ITask) => void, 
    onError: (err: any) => void,
    onConnect?: () => void
) => {
    const token = localStorage.getItem('token');
    const ctrl = new AbortController();

    // SSE Эндпоинт: GET /api/v1/tasks/subscribe
    fetchEventSource(`${import.meta.env.VITE_API_URL}/tasks/subscribe`, {
        method: 'GET',
        headers: {
            Authorization: `Bearer ${token}`,
        },
        signal: ctrl.signal,
        onmessage(ev) {
            if (ev.event === 'connected') {
                if (onConnect) onConnect();
                console.log('SSE Connected');
                return;
            }

            if (ev.data) {
                try {
                    const updatedTask = JSON.parse(ev.data);
                    onMessage(updatedTask);
                } catch (e) {
                    console.error("SSE Parse Error", e);
                }
            }
        },
        onerror(err) {
            onError(err);
        }
    });

    return () => ctrl.abort();
};