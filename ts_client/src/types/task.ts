// Описание одной задачи
export interface ITask {
    id: string;
    name: string;
    type: 'ACTUALIZE_BY_ID' | 'ACTUALIZE_ACTIVE' | 'FIND_NEW';
    status: 'pending' | 'running' | 'completed' | 'failed';
    created_at: string;
    started_at?: string;
    finished_at?: string;
    
    // Результаты (обновляются в реальном времени)
    result_summary?: {
        expected_results_count: number; // Сколько всего ожидаем (100%)
        total_processed: number;        // Сколько сделали
        created: number;
        updated: number;
        archived: number;
        new_links_found: number;
        id?: string;
    };
    created_by_user_id: string;
}

// Ответ списка задач (с пагинацией, как у вас в JSON)
export interface ITasksResponse {
    data: ITask[];
    total: number;
    page: number;
    perPage: number;
}