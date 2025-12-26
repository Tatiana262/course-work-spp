import React from 'react';
import { Modal, Button, Row, Col, Card, Badge, Accordion } from 'react-bootstrap';
import type { ITask } from '../../types/task';

interface Props {
    show: boolean;
    onHide: () => void;
    task: ITask | null;
}

const TaskDetailsModal: React.FC<Props> = ({ show, onHide, task }) => {
    if (!task) return null;

    const summary = task.result_summary;

    // Хелпер для форматирования даты
    const formatDate = (dateStr?: string) => {
        if (!dateStr) return '-';
        return new Date(dateStr).toLocaleString();
    };

    // Хелпер для расчета длительности
    const getDuration = () => {
        if (!task.started_at || !task.finished_at) return null;
        const start = new Date(task.started_at).getTime();
        const end = new Date(task.finished_at).getTime();
        const diffMs = end - start;
        const seconds = Math.floor(diffMs / 1000);
        
        if (seconds < 60) return `${seconds} сек`;
        const minutes = Math.floor(seconds / 60);
        return `${minutes} мин ${seconds % 60} сек`;
    };

    const duration = getDuration();

    // Перевод статуса и типа
    const getStatusBadge = (status: string) => {
        const map: any = {
            'completed': <Badge bg="success">Завершено</Badge>,
            'running': <Badge bg="primary">Выполняется</Badge>,
            'failed': <Badge bg="danger">Ошибка</Badge>,
            'pending': <Badge bg="secondary">В очереди</Badge>
        };
        return map[status] || <Badge bg="light" text="dark">{status}</Badge>;
    };

    const translateType = (type: string) => {
        const map: Record<string, string> = {
            'FIND_NEW': 'Поиск новых объектов',
            'ACTUALIZE_ACTIVE': 'Актуализация активных',
            'ACTUALIZE_ARCHIVED': 'Актуализация архивных',
            'ACTUALIZE_BY_ID': 'Актуализация одного объекта'
        };
        return map[type] || type;
    };

    return (
        <Modal show={show} onHide={onHide} size="lg" centered>
            <Modal.Header closeButton className="bg-light">
                <Modal.Title>
                    <span className="me-2">Детали задачи</span>
                    <span style={{ fontSize: '0.6em', opacity: 0.7 }}>#{task.id}</span>
                </Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {/* --- БЛОК 1: ОСНОВНОЕ --- */}
                <div className="d-flex justify-content-between align-items-start mb-4">
                    <div>
                        <h5 className="mb-1">{task.name}</h5>
                        <div className="text-muted small">{translateType(task.type)}</div>
                    </div>
                    <div className="text-end">
                        <div className="mb-1">{getStatusBadge(task.status)}</div>
                        {duration && <div className="text-muted small">Время: {duration}</div>}
                    </div>
                </div>

                {/* --- БЛОК 2: СТАТИСТИКА (Карточки) --- */}
                {summary && (
                    <div className="mb-4">
                        <h6 className="text-uppercase text-muted small fw-bold mb-3">Результаты выполнения</h6>
                        <Row className="g-3">
                            {/* Всего обработано */}
                            <Col xs={6} md={3}>
                                <Card className="h-100 text-center border-light bg-light">
                                    <Card.Body className="p-2">
                                        <div className="text-muted small text-uppercase">Обработано</div>
                                        <div className="fs-4 fw-bold">
                                            {summary.total_processed}
                                        </div>
                                    </Card.Body>
                                </Card>
                            </Col>

                            {/* Новые (только для FIND_NEW) */}
                            {task.type === 'FIND_NEW' && (
                                <Col xs={6} md={3}>
                                    <Card className="h-100 text-center border-success bg-opacity-10" style={{backgroundColor: '#f0fff4'}}>
                                        <Card.Body className="p-2">
                                            <div className="text-success small text-uppercase fw-bold">Найдено ссылок</div>
                                            <div className="fs-4 fw-bold text-success">
                                                +{summary.new_links_found}
                                            </div>
                                        </Card.Body>
                                    </Card>
                                </Col>
                            )}

                            {/* Создано */}
                            <Col xs={6} md={task.type === 'FIND_NEW' ? 2 : 3}>
                                <Card className="h-100 text-center border-success">
                                    <Card.Body className="p-2">
                                        <div className="text-success small text-uppercase">Создано</div>
                                        <div className="fs-4 fw-bold">{summary.created}</div>
                                    </Card.Body>
                                </Card>
                            </Col>

                            {/* Обновлено */}
                            <Col xs={6} md={task.type === 'FIND_NEW' ? 2 : 3}>
                                <Card className="h-100 text-center border-primary">
                                    <Card.Body className="p-2">
                                        <div className="text-primary small text-uppercase">Обновлено</div>
                                        <div className="fs-4 fw-bold">{summary.updated}</div>
                                    </Card.Body>
                                </Card>
                            </Col>

                            {/* Архив */}
                            <Col xs={6} md={task.type === 'FIND_NEW' ? 2 : 3}>
                                <Card className="h-100 text-center border-secondary">
                                    <Card.Body className="p-2">
                                        <div className="text-secondary small text-uppercase">В архив</div>
                                        <div className="fs-4 fw-bold">{summary.archived}</div>
                                    </Card.Body>
                                </Card>
                            </Col>
                        </Row>
                    </div>
                )}

                {/* --- БЛОК 3: ВРЕМЕННЫЕ МЕТКИ --- */}
                <Card className="mb-4 border-0 bg-light">
                    <Card.Body className="py-2">
                        <Row className="text-center small">
                            <Col>
                                <div className="text-muted">Создана</div>
                                <div>{formatDate(task.created_at)}</div>
                            </Col>
                            <Col>
                                <div className="text-muted">Запущена</div>
                                <div>{formatDate(task.started_at)}</div>
                            </Col>
                            <Col>
                                <div className="text-muted">Завершена</div>
                                <div>{formatDate(task.finished_at)}</div>
                            </Col>
                        </Row>
                    </Card.Body>
                </Card>

                {/* --- БЛОК 4: ТЕХНИЧЕСКИЕ ДАННЫЕ (Accordion) --- */}
                <Accordion>
                    <Accordion.Item eventKey="0">
                        <Accordion.Header>Техническая информация (JSON)</Accordion.Header>
                        <Accordion.Body>
                            <pre className="small m-0" style={{ whiteSpace: 'pre-wrap', maxHeight: '300px', overflowY: 'auto' }}>
                                {JSON.stringify(task, null, 2)}
                            </pre>
                        </Accordion.Body>
                    </Accordion.Item>
                </Accordion>

            </Modal.Body>
            <Modal.Footer>
                <Button variant="secondary" onClick={onHide}>Закрыть</Button>
            </Modal.Footer>
        </Modal>
    );
};

export default TaskDetailsModal;