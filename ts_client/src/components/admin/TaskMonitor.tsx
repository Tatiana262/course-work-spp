import React, { useEffect, useState } from 'react';
import { Table, ProgressBar, Badge, Spinner, Button, Modal } from 'react-bootstrap';
import { fetchTasks, subscribeToTasks } from '../../http/adminAPI';
import type { ITask } from '../../types/task';

const TaskMonitor = () => {
    const [tasks, setTasks] = useState<ITask[]>([]);
    const [loading, setLoading] = useState(true);
    const [connected, setConnected] = useState(false);

    // Модалка для деталей (JSON view)
    const [selectedTask, setSelectedTask] = useState<ITask | null>(null);

    useEffect(() => {
        loadTasks();
    }, []);

    const loadTasks = () => {
        fetchTasks(1, 20)
            .then(response => {
                // Сортировка: запущенные выше, потом по дате
                setTasks(response.data); 
            })
            .catch(console.error)
            .finally(() => setLoading(false));
    };

    // Подписка SSE
    useEffect(() => {
        const unsubscribe = subscribeToTasks(
            (updatedTask) => {
                if (!updatedTask || !updatedTask.id) return;

                setConnected(true);
                setTasks(prev => {
                    const index = prev.findIndex(t => t.id === updatedTask.id);
                    if (index !== -1) {
                        const newArr = [...prev];
                        newArr[index] = updatedTask;
                        return newArr;
                    } else {
                        // Если новая задача - добавляем в начало
                        return [updatedTask, ...prev];
                    }
                });
            },
            (err) => {
                console.error("SSE Error:", err);
                setConnected(false);
                // Если соединение разорвалось, можно попробовать перезапросить список через 5 сек
            },
            () => {
                // onConnect (НОВЫЙ КОЛБЭК)
                setConnected(true); // Ставим Live сразу при подключении!
            }
        );
        return () => {
            console.log("Disconnecting SSE...");
            unsubscribe();
        };
    }, []);

    // --- ЛОГИКА ПРОГРЕССА ---
    const calculateProgress = (task: ITask) => {
        if (task.status === 'completed') return 100;
        if (!task.result_summary) return 0;
        
        let processed: number, expected: number 
        if (task.type === 'FIND_NEW') {
            const { total_processed, new_links_found } = task.result_summary;
            processed = total_processed
            expected = new_links_found
        } else {
            const { total_processed, expected_results_count } = task.result_summary;
            processed = total_processed
            expected = expected_results_count
        }
        
        if (!expected || expected <= 0) return 0; // Защита от деления на 0
        
        const percent = (processed / expected) * 100;
        return Math.min(Math.round(percent), 100); // Не больше 100%
    };

    const getStatusBadge = (status: string) => {
        const map: any = {
            'completed': <Badge bg="success">Готово</Badge>,
            'running': <Badge bg="primary">В работе</Badge>,
            'failed': <Badge bg="danger">Ошибка</Badge>,
            'pending': <Badge bg="secondary">В обработке</Badge>
        };
        return map[status] || <Badge bg="light" text="dark">{status}</Badge>;
    };

    return (
        <div>
            <div className="d-flex justify-content-between align-items-center mb-3">
                <h5>Последние задачи</h5>
                <div>
                    Статус: {connected ? <Badge bg="success">Live</Badge> : <Badge bg="secondary">Offline</Badge>}
                    <Button variant="link" size="sm" onClick={loadTasks} className="ms-2">
                        <i className="bi bi-arrow-clockwise"></i> Обновить вручную
                    </Button>
                </div>
            </div>
            
            {/* ТАБЛИЦА С ЗАДАЧАМИ (без изменений в структуре) */}
            {loading ? <Spinner animation="border" /> : (
                <Table striped bordered hover responsive size="sm">
                    <thead>
                        <tr>
                            <th>Задача</th>
                            <th style={{width: '100px'}}>Статус</th>
                            <th style={{width: '25%'}}>Прогресс</th>
                            <th>Результат</th>
                            <th>Время</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {tasks.map(task => {
                            const progress = calculateProgress(task);
                            const summary = task.result_summary;

                            return (
                                <tr key={task.id}>
                                    <td>
                                        <div className="fw-bold small">{task.name || task.type}</div>
                                        <div className="text-muted" style={{fontSize: '0.7rem'}}>{task.id}</div>
                                    </td>
                                    <td>
                                        {/* Ваш badge */}
                                        <Badge bg={task.status === 'completed' ? 'success' : task.status === 'running' ? 'primary' : task.status === 'failed' ? 'danger' : 'secondary'}>
                                            {task.status}
                                        </Badge>
                                    </td>
                                    <td>
                                        {task.status === 'running' ? (
                                            <ProgressBar animated now={progress} label={`${progress}%`} variant={progress < 100 ? "info" : "success"} />
                                        ) : task.status === 'completed' ? (
                                            <ProgressBar variant="success" now={100} label="100%" />
                                        ) : (
                                            <span className="text-muted small">Ожидание...</span>
                                        )}
                                    </td>
                                    <td className="small">
                                        {summary && (
                                            <div style={{fontSize: '0.75rem'}}>
                                                {summary.created > 0 && <div className="text-success">+{summary.created} новыx</div>}
                                                {summary.updated > 0 && <div className="text-primary">~{summary.updated} обн.</div>}
                                                {/* Защита от undefined */}
                                                {task.type === 'FIND_NEW' 
                                                    ? `${summary.total_processed} (Новых: ${summary.new_links_found})` 
                                                    : `${summary.total_processed} / ${summary.expected_results_count}`
                                                }
                                              
                                            </div>
                                        )}
                                    </td>
                                    <td className="small text-muted">
                                        {new Date(task.created_at).toLocaleTimeString()}
                                    </td>
                                    <td>
                                        <Button size="sm" variant="outline-secondary" onClick={() => setSelectedTask(task)} style={{fontSize: '0.7rem'}}>
                                            JSON
                                        </Button>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </Table>
            )}
            
            <Modal show={!!selectedTask} onHide={() => setSelectedTask(null)} size="lg">
                <Modal.Header closeButton><Modal.Title>Детали</Modal.Title></Modal.Header>
                <Modal.Body>
                    <pre className="bg-light p-2 small">{JSON.stringify(selectedTask, null, 2)}</pre>
                </Modal.Body>
            </Modal>
        </div>
    );
};

export default TaskMonitor;