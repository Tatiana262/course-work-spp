import React, { useState, useEffect } from 'react';
import { Card, Button, Form, Row, Col, Alert, Badge } from 'react-bootstrap';
import { findNewObjects, actualizeActive } from '../../http/adminAPI';
import { fetchDictionaries } from '../../http/filterAPI';

const AdminControls = () => {
    const [dictionaries, setDictionaries] = useState<any>({});
    
    const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
    const [selectedRegions, setSelectedRegions] = useState<string[]>([]);
    
    const [actCategory, setActCategory] = useState('apartment');

    const [msg, setMsg] = useState<{type: string, text: string} | null>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        fetchDictionaries().then(data => {
            setDictionaries(data);
            if (data.categories && data.categories.length > 0) {
                setActCategory(data.categories[0].system_name);
            }
        });
    }, []);

    // Хелпер для переключения чекбокса (добавить/удалить из массива)
    const toggleItem = (list: string[], setList: (l: string[]) => void, value: string) => {
        if (list.includes(value)) {
            setList(list.filter(item => item !== value));
        } else {
            setList([...list, value]);
        }
    };

    // --- ЗАПУСК ПОИСКА НОВЫХ ---
    const handleFindNew = async () => {
        if (selectedCategories.length === 0 || selectedRegions.length === 0) {
            setMsg({ type: 'warning', text: 'Выберите хотя бы одну категорию и регион!' });
            return;
        }

        console.log("Отправляем на сервер:", { 
            categories: selectedCategories, 
            regions: selectedRegions 
        });
        
        setLoading(true);
        try {
            // Отправляем массивы!
            await findNewObjects(selectedCategories, selectedRegions);
            setMsg({ 
                type: 'success', 
                text: `Задача создана! Категорий: ${selectedCategories.length}, Регионов: ${selectedRegions.length}` 
            });
            // Очистка формы (по желанию)
            // setSelectedCategories([]);
            // setSelectedRegions([]);
        } catch (e) {
            console.error(e);
            setMsg({ type: 'danger', text: 'Ошибка при запуске задачи' });
        } finally {
            setLoading(false);
        }
    };

    // --- ЗАПУСК АКТУАЛИЗАЦИИ ---
    const handleActualizeActive = async () => {
        if (!window.confirm(`Проверить ВСЕ активные объекты в категории "${actCategory}"?`)) return;
        
        setLoading(true);
        try {
            await actualizeActive(actCategory, 100); 
            setMsg({ type: 'success', text: `Актуализация запущена для: ${actCategory}` });
        } catch (e) {
            setMsg({ type: 'danger', text: 'Ошибка запуска' });
        } finally {
            setLoading(false);
        }
    };

    return (
        <div>
            {msg && <Alert variant={msg.type as any} onClose={() => setMsg(null)} dismissible>{msg.text}</Alert>}

            <Row>
                {/* === КАРТОЧКА 1: ПОИСК НОВЫХ (Множественный выбор) === */}
                <Col lg={7}>
                    <Card className="mb-4 shadow-sm h-100">
                        <Card.Header className="bg-primary text-white d-flex justify-content-between align-items-center">
                            <span>Поиск новых объявлений</span>
                            <Badge bg="light" text="dark">Multi</Badge>
                        </Card.Header>
                        <Card.Body>
                            <Form>
                                <Row>
                                    {/* Колонка КАТЕГОРИИ */}
                                    <Col sm={6}>
                                        <Form.Label className="fw-bold">Категории</Form.Label>
                                        <div className="border rounded p-2 mb-3" style={{ maxHeight: '200px', overflowY: 'auto' }}>
                                            {dictionaries.categories?.map((c: any) => {
                                                // Используем то значение, которое ждет бэкенд (system_name или display_name)
                                                // Предположим, вы отправляете display_name ("Квартиры") как в JSON примере
                                                return (
                                                    <Form.Check 
                                                        key={c.system_name}
                                                        type="checkbox"
                                                        label={c.display_name}
                                                        checked={selectedCategories.includes(c.system_name)}
                                                        onChange={() => toggleItem(selectedCategories, setSelectedCategories, c.system_name)}
                                                    />
                                                )
                                            })}
                                        </div>
                                    </Col>

                                    {/* Колонка РЕГИОНЫ */}
                                    <Col sm={6}>
                                        <Form.Label className="fw-bold">Регионы</Form.Label>
                                        <div className="border rounded p-2 mb-3" style={{ maxHeight: '200px', overflowY: 'auto' }}>
                                            {dictionaries.regions?.map((r: any) => {
                                                
                                                return (
                                                    <Form.Check 
                                                        key={r.system_name}
                                                        type="checkbox"
                                                        label={r.display_name}
                                                        checked={selectedRegions.includes(r.system_name)}
                                                        onChange={() => toggleItem(selectedRegions, setSelectedRegions, r.system_name)}
                                                    />
                                                )
                                            })}
                                            
                                        </div>
                                    </Col>
                                </Row>

                                <div className="d-flex gap-2 justify-content-end">
                                    <Button 
                                        variant="outline-secondary" 
                                        size="sm"
                                        onClick={() => {setSelectedCategories([]); setSelectedRegions([])}}
                                    >
                                        Сброс
                                    </Button>
                                    <Button 
                                        variant="success" 
                                        onClick={handleFindNew}
                                        disabled={loading || selectedCategories.length === 0 || selectedRegions.length === 0}
                                    >
                                        {loading ? 'Запуск...' : 'Найти новые'}
                                    </Button>
                                </div>
                            </Form>
                        </Card.Body>
                    </Card>
                </Col>

                {/* === КАРТОЧКА 2: АКТУАЛИЗАЦИЯ (Одиночный выбор для безопасности) === */}
                <Col lg={5}>
                    <Card className="mb-4 shadow-sm h-100">
                        <Card.Header className="bg-warning text-dark">Актуализация базы</Card.Header>
                        <Card.Body>
                            <Card.Text className="small text-muted">
                                Проверяет, изменилась ли цена или статус (Active/Archived) у существующих объявлений.
                            </Card.Text>
                            
                            <Form.Group className="mb-3">
                                <Form.Label>Выберите категорию:</Form.Label>
                                <Form.Select 
                                    value={actCategory} 
                                    onChange={e => setActCategory(e.target.value)}
                                >
                                    {dictionaries.categories?.map((c: any) => (
                                        <option key={c.system_name} value={c.system_name}>{c.display_name}</option>
                                    ))}
                                </Form.Select>
                            </Form.Group>

                            <div className="d-grid gap-2">
                                <Button 
                                    variant="outline-dark" 
                                    onClick={handleActualizeActive}
                                    disabled={loading}
                                >
                                    {loading ? 'Работаем...' : 'Запустить проверку'}
                                </Button>
                            </div>
                        </Card.Body>
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default AdminControls;