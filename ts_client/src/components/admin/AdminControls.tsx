import React, { useState, useEffect, useMemo } from 'react';
import { Card, Button, Form, Row, Col, Alert, Badge, InputGroup } from 'react-bootstrap';
import { findNewObjects, actualizeActive, actualizeArchived, fetchStats } from '../../http/adminAPI';
import { fetchDictionaries } from '../../http/filterAPI';
import type { IStatItem } from "../../types/actualization";
import type { IDictionariesResponse } from '../../types/filter';

const AdminControls = () => {
    // --- STATE ---
    const [dictionaries, setDictionaries] = useState<any>({});
    const [stats, setStats] = useState<IStatItem[]>([]);
    
    // Поиск новых
    const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
    const [selectedRegions, setSelectedRegions] = useState<string[]>([]);
    
    // Актуализация: Инициализируем пустой строкой ("Все")
    const [actActiveCategory, setActActiveCategory] = useState('');
    const [actActiveLimit, setActActiveLimit] = useState(10);

    const [actArchivedCategory, setActArchivedCategory] = useState('');
    const [actArchivedLimit, setActArchivedLimit] = useState(10);

    const [msg, setMsg] = useState<{type: string, text: string} | null>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        fetchDictionaries()
            .then(data => setDictionaries(data))
            .catch(err => console.error("Ошибка загрузки словарей", err));
    }, []);

    // 2. ФУНКЦИЯ ЗАГРУЗКИ СТАТИСТИКИ
    // Вынесли отдельно, чтобы вызывать из разных мест
    const loadStatsData = () => {
        fetchStats()
            .then(data => setStats(data))
            .catch(err => console.error("Ошибка загрузки статистики", err));
    };

    // 3. ЭФФЕКТ ДЛЯ СТАТИСТИКИ (Mount + Focus)
    useEffect(() => {
        // Загружаем сразу при входе
        loadStatsData();

        // Обновляем при возвращении на вкладку
        const onFocus = () => {
            // Можно добавить проверку !document.hidden или просто вызывать
            // console.log("AdminControls focused, updating stats...");
            loadStatsData();
        };

        window.addEventListener("focus", onFocus);

        return () => {
            window.removeEventListener("focus", onFocus);
        };
    }, []);

    const getCategoryName = (sysName: string) => {
        if (!sysName) return 'Все категории';
        const cat = dictionaries.categories?.find((c: any) => c.system_name === sysName);
        return cat ? cat.display_name : sysName;
    };

    const toggleItem = (list: string[], setList: (l: string[]) => void, value: string) => {
        if (list.includes(value)) setList(list.filter(item => item !== value));
        else setList([...list, value]);
    };

    const handleSelectAll = (allItems: any[], currentList: string[], setList: (l: string[]) => void) => {
        if (!allItems) return;
        const allIds = allItems.map(i => i.system_name);
        if (currentList.length === allIds.length) setList([]);
        else setList(allIds);
    };

    const handleFindNew = async () => {
        if (selectedCategories.length === 0 || selectedRegions.length === 0) {
            setMsg({ type: 'warning', text: 'Выберите категории и регионы!' });
            return;
        }
        setLoading(true);
        try {
            await findNewObjects(selectedCategories, selectedRegions);
            setMsg({ type: 'success', text: `Задача на поиск создана!` });
        } catch (e) {
            setMsg({ type: 'danger', text: 'Ошибка запуска поиска' });
        } finally {
            setLoading(false);
        }
    };

    const handleActualize = async (type: 'active' | 'archived') => {
        setLoading(true);
        try {
            if (type === 'active') {
                await actualizeActive(actActiveCategory, actActiveLimit);
                const catName = getCategoryName(actActiveCategory);
                setMsg({ type: 'success', text: `Актуализация активных запущена (${catName}, ${actActiveLimit} шт)` });
            } else {
                await actualizeArchived(actArchivedCategory, actArchivedLimit);
                const catName = getCategoryName(actArchivedCategory);
                setMsg({ type: 'success', text: `Актуализация архивных запущена (${catName}, ${actActiveLimit} шт)` });
            }
        } catch (e) {
            setMsg({ type: 'danger', text: 'Ошибка запуска актуализации' });
        } finally {
            setLoading(false);
        }
    };

    // --- ЛОГИКА СТАТИСТИКИ ---
    // Вычисляем статистику для выбранной категории или суммируем для "Всех"
    const getStat = (categorySysName: string) => {
        if (categorySysName === "") {
            // Если выбрано "Все категории", суммируем показатели
            return stats.reduce((acc, item) => ({
                display_name: "Все",
                system_name: "",
                active_count: acc.active_count + item.active_count,
                archived_count: acc.archived_count + item.archived_count
            }), { active_count: 0, archived_count: 0 } as IStatItem);
        }
        return stats.find(s => s.system_name === categorySysName);
    };

    return (
        <div>
            {msg && <Alert variant={msg.type as any} onClose={() => setMsg(null)} dismissible>{msg.text}</Alert>}

            {/* Секция 1: ПОИСК НОВЫХ (без изменений) */}
            <Card className="mb-4 shadow-sm border-primary">
                {/* ... (код поиска новых такой же, как раньше) ... */}
                <Card.Header className="bg-primary text-white d-flex justify-content-between align-items-center">
                    <span className="fw-bold">1. Поиск новых объявлений</span>
                </Card.Header>
                <Card.Body>
                    <Row>
                        <Col md={6} className="mb-3">
                            <div className="d-flex justify-content-between mb-2">
                                <Form.Label className="fw-bold m-0">Категории</Form.Label>
                                <Form.Check 
                                    type="checkbox" label="Все" 
                                    checked={dictionaries.categories && selectedCategories.length === dictionaries.categories.length}
                                    onChange={() => handleSelectAll(dictionaries.categories, selectedCategories, setSelectedCategories)}
                                />
                            </div>
                            <div className="border rounded p-2" style={{ maxHeight: '150px', overflowY: 'auto' }}>
                                {dictionaries.categories?.map((c: any) => (
                                    <Form.Check 
                                        key={c.system_name} type="checkbox" label={c.display_name}
                                        checked={selectedCategories.includes(c.system_name)}
                                        onChange={() => toggleItem(selectedCategories, setSelectedCategories, c.system_name)}
                                    />
                                ))}
                            </div>
                        </Col>
                        <Col md={6} className="mb-3">
                            <div className="d-flex justify-content-between mb-2">
                                <Form.Label className="fw-bold m-0">Регионы</Form.Label>
                                <Form.Check 
                                    type="checkbox" label="Все" 
                                    checked={dictionaries.regions && selectedRegions.length === dictionaries.regions.length}
                                    onChange={() => handleSelectAll(dictionaries.regions, selectedRegions, setSelectedRegions)}
                                />
                            </div>
                            <div className="border rounded p-2" style={{ maxHeight: '150px', overflowY: 'auto' }}>
                                {dictionaries.regions?.map((r: any) => (
                                    <Form.Check 
                                        key={r.system_name} type="checkbox" label={r.display_name}
                                        checked={selectedRegions.includes(r.system_name)}
                                        onChange={() => toggleItem(selectedRegions, setSelectedRegions, r.system_name)}
                                    />
                                ))}
                            </div>
                        </Col>
                    </Row>
                    <div className="text-end">
                        <Button variant="primary" onClick={handleFindNew} disabled={loading}>{loading ? 'Запуск...' : 'Найти новые объекты'}</Button>
                    </div>
                </Card.Body>
            </Card>

            {/* === СЕКЦИЯ 2: АКТУАЛИЗАЦИЯ === */}
            <Row>
                {/* АКТИВНЫЕ */}
                <Col md={6}>
                    <Card className="mb-4 shadow-sm border-warning h-100">
                        <Card.Header className="bg-warning text-dark fw-bold">2. Актуализация активных</Card.Header>
                        <Card.Body>
                            <Form.Group className="mb-3">
                                <Form.Label>Категория</Form.Label>
                                <Form.Select value={actActiveCategory} onChange={e => setActActiveCategory(e.target.value)}>
                                    {/* ПУНКТ "ВСЕ КАТЕГОРИИ" */}
                                    <option value="">Все категории</option>
                                    
                                    {dictionaries.categories?.map((c: any) => (
                                        <option key={c.system_name} value={c.system_name}>{c.display_name}</option>
                                    ))}
                                </Form.Select>
                                
                                <div className="mt-2 small text-muted">
                                    В базе: <strong>{getStat(actActiveCategory)?.active_count || 0}</strong> активных
                                </div>
                            </Form.Group>

                            <Form.Group className="mb-3">
                                <Form.Label>Лимит (сколько проверить)</Form.Label>
                                <InputGroup>
                                    <Form.Control type="number" value={actActiveLimit} onChange={e => setActActiveLimit(Number(e.target.value))} min={1} />
                                    <Button variant="outline-warning text-dark" onClick={() => handleActualize('active')} disabled={loading}>Запустить</Button>
                                </InputGroup>
                            </Form.Group>
                        </Card.Body>
                    </Card>
                </Col>

                {/* АРХИВНЫЕ */}
                <Col md={6}>
                    <Card className="mb-4 shadow-sm border-secondary h-100">
                        <Card.Header className="bg-secondary text-white fw-bold">3. Актуализация архивных</Card.Header>
                        <Card.Body>
                            <Form.Group className="mb-3">
                                <Form.Label>Категория</Form.Label>
                                <Form.Select value={actArchivedCategory} onChange={e => setActArchivedCategory(e.target.value)}>
                                    {/* ПУНКТ "ВСЕ КАТЕГОРИИ" */}
                                    <option value="">Все категории</option>

                                    {dictionaries.categories?.map((c: any) => (
                                        <option key={c.system_name} value={c.system_name}>{c.display_name}</option>
                                    ))}
                                </Form.Select>
                                
                                <div className="mt-2 small text-muted">
                                    В архиве: <strong>{getStat(actArchivedCategory)?.archived_count || 0}</strong> объектов
                                </div>
                            </Form.Group>

                            <Form.Group className="mb-3">
                                <Form.Label>Лимит (сколько проверить)</Form.Label>
                                <InputGroup>
                                    <Form.Control type="number" value={actArchivedLimit} onChange={e => setActArchivedLimit(Number(e.target.value))} min={1} />
                                    <Button variant="outline-secondary" onClick={() => handleActualize('archived')} disabled={loading}>Запустить</Button>
                                </InputGroup>
                            </Form.Group>
                        </Card.Body>
                    </Card>
                </Col>
            </Row>
        </div>
    );
};

export default AdminControls;