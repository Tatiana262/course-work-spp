import React, { useEffect, useState, useMemo } from 'react';
import { Form, Button, Row, Col, Card, Modal, Badge, Spinner, Dropdown } from 'react-bootstrap';
import { useSearchParams } from 'react-router-dom'; // <--- Добавили хук
import type { IFilterState, IFilterOptionsResponse } from '../types/filter';
import { fetchFilterOptions, fetchDictionaries } from '../http/filterAPI';
import { INITIAL_FILTERS, ARRAY_KEYS } from '../utils/filterUtils'; 
import useDebounce from '../hooks/useDebounce';

interface Props {
    onSearch: (filters: IFilterState) => void;
}

const AdvancedFilterBar: React.FC<Props> = ({ onSearch }) => {
    // --- STATE ---
    const [dictionaries, setDictionaries] = useState<any>({});
    const [dynamicOptions, setDynamicOptions] = useState<IFilterOptionsResponse | null>(null);
    const [showModal, setShowModal] = useState(false);
    const [loadingOptions, setLoadingOptions] = useState(false);
    
    // Хук для работы с URL
    const [searchParams, setSearchParams] = useSearchParams();

    // Инициализируем стейт. 
    // ВАЖНО: Если мы уже на странице Listings, параметры могут быть в URL сразу при загрузке.
    // Мы можем инициализировать стейт лениво (функцией), чтобы сразу прочитать URL.
    const [filters, setFilters] = useState<IFilterState>(() => {
        // Проверяем, есть ли параметры в URL
        if ([...searchParams].length > 0) {
            const newFilters = { ...INITIAL_FILTERS };
            
            searchParams.forEach((value, key) => {
                 // @ts-ignore
                if (key in newFilters) {
                     // @ts-ignore
                    if (ARRAY_KEYS.includes(key)) {
                        const val = searchParams.get(key);
                        if (val) {
                            // @ts-ignore
                            newFilters[key] = val.split(',');
                        }
                    } else {
                        // Если строка
                        // @ts-ignore
                        newFilters[key] = value;
                    }
                }
            });
            return newFilters;
        }
        return INITIAL_FILTERS;
    });

    // useEffect(() => {
    //     const paramsCount = [...searchParams].length;
    //     if (paramsCount > 0) {
    //          const newFilters = { ...INITIAL_FILTERS };
             
    //          searchParams.forEach((value, key) => {
    //             // @ts-ignore
    //             if (key in newFilters) {
    //                  // @ts-ignore
    //                 if (ARRAY_KEYS.includes(key)) {
    //                     // @ts-ignore
    //                     newFilters[key] = searchParams.getAll(key);
    //                 } else {
    //                     // @ts-ignore
    //                     newFilters[key] = value;
    //                 }
    //             }
    //         });

    //         setFilters(newFilters);
    //         // // --- ГЛАВНОЕ ИСПРАВЛЕНИЕ ---
    //         // // Мы сравниваем текущие фильтры и новые (пришедшие из URL).
    //         // // Используем JSON.stringify для простого глубокого сравнения объектов.
    //         // // Если они равны — значит изменилась только страница (page=2), 
    //         // // и нам НЕ НАДО обновлять стейт и вызывать лишний запрос.
    //         // if (JSON.stringify(newFilters) !== JSON.stringify(filters)) {
    //         //     setFilters(newFilters);
    //         // }
    //     }
    // }, [searchParams]);

    // Инициализируем стейт начальным значением
    // const [filters, setFilters] = useState<IFilterState>(INITIAL_FILTERS);

    const debouncedFilters = useDebounce(filters, 500);

    // --- EFFECTS ---

    // 1. Загрузка словарей (один раз)
    useEffect(() => {
        fetchDictionaries().then(setDictionaries);
    }, []);

    // 2. СИНХРОНИЗАЦИЯ С URL ПРИ ЗАГРУЗКЕ
    // Это решает проблему сохранения состояния при переходе на страницу списка
   useEffect(() => {
        // Этот эффект сработает, если URL изменился, а компонент не пересоздался
        const paramsCount = [...searchParams].length;
        if (paramsCount > 0) {
             const newFilters = { ...INITIAL_FILTERS };
             let hasChanges = false;

             searchParams.forEach((value, key) => {
                if (key === 'page') return;
                
                 // @ts-ignore
                if (key in newFilters) {
                    hasChanges = true;
                     // @ts-ignore
                    if (ARRAY_KEYS.includes(key)) {
                        const val = searchParams.get(key);
                        if (val) {
                            // @ts-ignore
                            newFilters[key] = val.split(',');
                        }
                    } else {
                        // @ts-ignore
                        newFilters[key] = value;
                    }
                }
            });
            
            if (hasChanges) {
                // Чтобы не зациклить, можно добавить проверку deepEqual, 
                // но пока просто обновляем, так как searchParams меняется редко
                setFilters(newFilters);
            }
        }
    }, [searchParams]);

    // 3. Запрос опций при изменении фильтров
    useEffect(() => {
        setLoadingOptions(true);

        // --- ИСПРАВЛЕНИЕ ---
        // Преобразуем стейт перед отправкой: массивы склеиваем в строку через запятую
        const params: any = {};
        
        Object.entries(debouncedFilters).forEach(([key, value]) => {
            // Пропускаем пустые значения
            if (value === '' || value === null || (Array.isArray(value) && value.length === 0)) {
                return;
            }

            if (Array.isArray(value)) {
                // Превращаем ["Кирпич", "Панель"] в "Кирпич,Панель"
                params[key] = value.join(','); 
            } else {
                params[key] = value;
            }
        });

        fetchFilterOptions(params)
            .then(data => setDynamicOptions(data))
            .catch(console.error)
            .finally(() => setLoadingOptions(false));
            
    }, [debouncedFilters]);

    // При смене региона сбрасываем город
    const handleRegionChange = (val: string) => {
        setFilters(prev => ({ ...prev, region: val, cityOrDistrict: '' }));
    };

    const handleChange = (key: keyof IFilterState, val: string) => {
        setFilters(prev => ({ ...prev, [key]: val }));
    };

    // Исправленная логика множественного выбора (Пункт 4)
    // Приводим всё к строке, чтобы избежать проблем "1" vs 1
   const handleCheckboxChange = (key: keyof IFilterState, option: string | number) => {
        const valStr = String(option);
        setFilters(prev => {
            const currentArr = (prev[key] as string[]) || []; 
            if (currentArr.includes(valStr)) {
                return { ...prev, [key]: currentArr.filter(i => i !== valStr) };
            } else {
                return { ...prev, [key]: [...currentArr, valStr] };
            }
        });
    };

    // 4. СБРОС ФИЛЬТРОВ
    const handleReset = () => {
        // Сбрасываем к начальному состоянию
        setFilters(INITIAL_FILTERS);
        // Можно сразу вызвать поиск, чтобы очистить выдачу
        // onSearch(INITIAL_FILTERS); 
        // Или просто закрыть модалку, оставив пользователю возможность нажать "Найти"
    };

    const priceRange = dynamicOptions?.filters?.price; 
    const priceMinPlaceholder = priceRange?.min !== undefined ? `от ${priceRange.min}` : 'от';
    const priceMaxPlaceholder = priceRange?.max !== undefined ? `до ${priceRange.max}` : 'до';
    // --- RENDER HELPERS ---

    // Пункт 3: Плейсхолдеры из данных бэкенда
    const renderRangeInput = (label: string, backendKey: string, minKey: keyof IFilterState, maxKey: keyof IFilterState) => {
        const opt = dynamicOptions?.filters[backendKey];
        if (!opt) return null;
        return (
            <Form.Group className="mb-3">
                <Form.Label>{label}</Form.Label>
                <div className="d-flex gap-2">
                    <Form.Control type="number" placeholder={opt.min !== undefined ? `от ${opt.min}` : 'от'} value={filters[minKey] as string} onChange={e => handleChange(minKey, e.target.value)} />
                    <Form.Control type="number" placeholder={opt.max !== undefined ? `до ${opt.max}` : 'до'} value={filters[maxKey] as string} onChange={e => handleChange(maxKey, e.target.value)} />
                </div>
            </Form.Group>
        );
    };

    const renderCheckboxGroup = (label: string, backendKey: string, stateKey: keyof IFilterState) => {
        const opt = dynamicOptions?.filters[backendKey];
        if (!opt || !opt.options || opt.options.length === 0) return null;
        return (
            <Form.Group className="mb-3">
                <Form.Label className="fw-bold d-block">{label}</Form.Label>
                <div className="d-flex flex-wrap gap-2">
                    {opt.options.map((o: any) => {
                        const valStr = String(o);
                        const isChecked = (filters[stateKey] as string[])?.includes(valStr);
                        return (
                             <Button key={valStr} variant={isChecked ? "primary" : "outline-secondary"} size="sm" onClick={() => handleCheckboxChange(stateKey, valStr)}>
                                {o}
                            </Button>
                        )
                    })}
                </div>
            </Form.Group>
        )
    };

    // Хелпер для Multiselect (Dropdown с чекбоксами)
    const renderMultiSelect = (label: string, backendKey: string, stateKey: keyof IFilterState) => {
        const opt = dynamicOptions?.filters[backendKey];
        if (!opt || !opt.options || opt.options.length === 0) return null;

        const selectedCount = (filters[stateKey] as string[])?.length || 0;
        
        return (
            <Form.Group className="mb-3">
                <Form.Label>{label}</Form.Label>
                <Dropdown>
                    <Dropdown.Toggle variant="outline-secondary" className="w-100 text-start d-flex justify-content-between align-items-center">
                        <span className="text-truncate">
                            {selectedCount > 0 ? `Выбрано: ${selectedCount}` : 'Не выбрано'}
                        </span>
                    </Dropdown.Toggle>

                    <Dropdown.Menu className="w-100 p-2" style={{ maxHeight: '200px', overflowY: 'auto' }}>
                        {opt.options.map((o: any) => {
                            const valStr = String(o);
                            const isChecked = (filters[stateKey] as string[])?.includes(valStr);
                            
                            return (
                                <Form.Check 
                                    key={valStr}
                                    type="checkbox"
                                    label={o}
                                    checked={isChecked}
                                    onChange={() => handleCheckboxChange(stateKey, valStr)}
                                    className="mb-1"
                                />
                            );
                        })}
                    </Dropdown.Menu>
                </Dropdown>
            </Form.Group>
        );
    };

    // --- LOGIC: Подсчет активных фильтров в модальном окне (Пункт 5 и вопрос про ошибку) ---
    const activeFiltersCount = useMemo(() => {
        let count = 0;
        const modalKeys: (keyof IFilterState)[] = [
            'totalAreaMin', 'totalAreaMax', 'kitchenAreaMin', 'kitchenAreaMax',
            'yearBuiltMin', 'yearBuiltMax', 'wallMaterials', 
            'floorMin', 'floorMax', 'floorBuildingMin', 'floorBuildingMax',
            'repairState', 'bathroomType', 'balconyType',
            'houseTypes', 'plotAreaMin', 'plotAreaMax', 
            'roofMaterials', 'waterConditions', 'heatingConditions', 
            'electricityConditions', 'sewageConditions', 'gazConditions'
        ];
        modalKeys.forEach(key => {
            const val = filters[key];
            if (Array.isArray(val)) {
                if (val.length > 0) count++;
            } else if (typeof val === 'string') {
                if (val.trim() !== '' && val !== '0') count++;
            }
        });
        return count;
    }, [filters]);

    return (
        <>
            <Card className="p-3 shadow-sm mb-4">
                <Form>
                    {/* Первая строка: Основные селекты */}
                    <Row className="g-2">
                        <Col md={2}>
                            <Form.Select value={filters.dealType} onChange={e => handleChange('dealType', e.target.value)}>
                                {dictionaries.deal_types?.map((c: any) => <option key={c.system_name} value={c.system_name}>{c.display_name}</option>)}
                            </Form.Select>
                        </Col>
                        <Col md={2}>
                            <Form.Select value={filters.category} onChange={e => handleChange('category', e.target.value)}>
                                {dictionaries.categories?.map((c: any) => <option key={c.system_name} value={c.system_name}>{c.display_name}</option>)}
                            </Form.Select>
                        </Col>
                        <Col md={2}>
                            <Form.Select value={filters.region} onChange={e => handleRegionChange(e.target.value)}>
                                <option value="">Вся Беларусь</option>
                                {dictionaries.regions?.map((c: any) => <option key={c.system_name} value={c.system_name}>{c.display_name}</option>)}
                            </Form.Select>
                        </Col>
                        
                        {/* Пункт 1: Город через Input + Datalist */}
                        <Col md={2}>
                            <Form.Control 
                                list="city-options" 
                                placeholder={filters.region ? "Город / Район" : "Выберите регион"}
                                value={filters.cityOrDistrict}
                                onChange={e => handleChange('cityOrDistrict', e.target.value)}
                                disabled={!filters.region}
                            />
                            {/* Скрытый список вариантов */}
                            <datalist id="city-options">
                                {dynamicOptions?.filters?.cities?.options?.map((city: any) => (
                                    <option key={city} value={city} />
                                ))}
                            </datalist>
                        </Col>

                        {/* Пункт 2: Поле ввода улицы */}
                        <Col md={4}>
                            <Form.Control 
                                // size="sm"
                                placeholder="Улица, поселок..."
                                value={filters.street}
                                onChange={e => handleChange('street', e.target.value)}
                            />
                        </Col>
                        
                    </Row>

                    {/* Вторая строка: Цена, Комнаты ИЛИ Тип, Улица */}
                    <Row className="mt-2 g-2 align-items-center">
                        <Col md={4}>
                            <div className="d-flex gap-1 align-items-center">
                                <span className="text-muted me-1">Цена:</span>
                                
                                <Form.Control size="sm" placeholder={priceMinPlaceholder} value={filters.priceMin} onChange={e => handleChange('priceMin', e.target.value)} />
                                <span className="text-muted">-</span>
                                <Form.Control size="sm" placeholder={priceMaxPlaceholder} value={filters.priceMax} onChange={e => handleChange('priceMax', e.target.value)} />
                                <Form.Select size="sm" style={{width: '80px'}} value={filters.priceCurrency} onChange={e => handleChange('priceCurrency', e.target.value)}>
                                    <option value="USD">USD</option>
                                    <option value="BYN">BYN</option>
                                    <option value="EUR">EUR</option>
                                </Form.Select>
                            </div>
                        </Col>
                        
                        <Col md={4}>
                             {filters.category === 'commercial' ? (
                                 // 1. ДЛЯ КОММЕРЦИИ: Одиночный Select "Вид объекта"
                                 <div className="d-flex gap-1 align-items-center">
                                     <span className="text-muted me-1">Вид:</span>
                                     <Form.Select 
                                        size="sm"
                                        // Берем первый элемент массива (так как стейт хранит массив) или пустую строку
                                        value={filters.commercialTypes?.[0] || ''}
                                        onChange={(e) => {
                                            const val = e.target.value;
                                            // При выборе перезаписываем массив одним значением
                                            setFilters(prev => ({
                                                ...prev,
                                                commercialTypes: val ? [val] : []
                                            }));
                                        }}
                                     >
                                         <option value="">Любой</option>
                                         {/* Берем опции из commercial_types (как в твоем JSON) */}
                                         {dynamicOptions?.filters?.commercial_types?.options?.map((opt: any) => (
                                             <option key={opt} value={opt}>{opt}</option>
                                         ))}
                                     </Form.Select>
                                 </div>
                             ) : (
                                 // 2. ДЛЯ ЖИЛОЙ (Квартиры/Дома): Кнопки комнат
                                 <div className="d-flex gap-1 align-items-center flex-wrap">
                                    <span className="text-muted me-1">Комнаты:</span>
                                    {/* Берем опции из rooms (если нет, по умолчанию 1-4) */}
                                    {(dynamicOptions?.filters?.rooms?.options || [1, 2, 3, 4]).map((r: any) => {
                                        const valStr = String(r);
                                        const isChecked = filters.rooms.includes(Number(valStr)) || (filters.rooms as any[]).includes(valStr);
                                        return (
                                            <Button 
                                                key={r} 
                                                size="sm"
                                                variant={isChecked ? "secondary" : "outline-light text-dark border"}
                                                onClick={() => handleCheckboxChange('rooms', r)}
                                            >
                                                {r}
                                            </Button>
                                        )
                                    })}
                                 </div>
                             )}
                        </Col>

                        {/* Пункт 4: Комнаты теперь динамические (с сервера) */}
                       

                        <Col md={2}>
                            <Button 
                                variant="outline-primary" 
                                className="w-100 position-relative"
                                onClick={() => setShowModal(true)}
                            >
                                Расширенный поиск
                                {activeFiltersCount > 0 && (
                                    <Badge bg="danger" className="position-absolute top-0 start-100 translate-middle rounded-circle">
                                        {activeFiltersCount}
                                    </Badge>
                                )}
                            </Button>
                        </Col>

                        {/* Кнопка Найти */}
                        <Col md={2}>
                             <Button variant="success" className="w-100" onClick={() => onSearch(filters)}>
                                {loadingOptions ? <Spinner size="sm" animation="border"/> : `Показать ${dynamicOptions?.count || 0}`}
                            </Button>
                        </Col>
                    </Row>
                </Form>
            </Card>

            {/* === МОДАЛЬНОЕ ОКНО === */}
            <Modal show={showModal} onHide={() => setShowModal(false)} size="lg" centered>
                <Modal.Header closeButton>
                    <Modal.Title>Все параметры</Modal.Title>
                </Modal.Header>
                <Modal.Body style={{ maxHeight: '70vh', overflowY: 'auto' }}>
                    <Form>
                        <h5 className="mb-3 text-primary">Общие характеристики</h5>
                       

                        {filters.category === 'apartment' && (
                            <>
                                <Row>
                                    <Col md={6}>{renderRangeInput("Площадь общая (м²)", "total_area", "totalAreaMin", "totalAreaMax")}</Col>
                                    <Col md={6}>{renderRangeInput("Площадь кухни (м²)", "kitchen_area", "kitchenAreaMin", "kitchenAreaMax")}</Col>
                                    <Col md={6}>{renderRangeInput("Год постройки", "year_built", "yearBuiltMin", "yearBuiltMax")}</Col>
                                    <Col md={12}>{renderMultiSelect("Материал стен", "wall_materials", "wallMaterials")}</Col>
                                </Row>
                                <hr />
                                <h5 className="mb-3 text-primary">Для квартир</h5>
                                <Row>
                                    <Col md={6}>{renderRangeInput("Этаж", "floor", "floorMin", "floorMax")}</Col>
                                    <Col md={6}>{renderRangeInput("Этажность дома", "building_floor", "floorBuildingMin", "floorBuildingMax")}</Col>
                                    <Col md={12}>{renderMultiSelect("Ремонт", "repair_states", "repairState")}</Col>
                                    <Col md={12}>{renderMultiSelect("Санузел", "bathroom_types", "bathroomType")}</Col>
                                    <Col md={12}>{renderMultiSelect("Балкон", "balcony_types", "balconyType")}</Col>
                                </Row>
                            </>
                        )}

                        {filters.category === 'house' && (
                            <>
                                <Row>
                                    <Col md={6}>{renderRangeInput("Площадь общая (м²)", "total_area", "totalAreaMin", "totalAreaMax")}</Col>
                                    <Col md={6}>{renderRangeInput("Площадь кухни (м²)", "kitchen_area", "kitchenAreaMin", "kitchenAreaMax")}</Col>
                                    <Col md={6}>{renderRangeInput("Год постройки", "year_built", "yearBuiltMin", "yearBuiltMax")}</Col>
                                    <Col md={12}>{renderMultiSelect("Материал стен", "wall_materials", "wallMaterials")}</Col>
                                </Row>
                                <hr />
                                <h5 className="mb-3 text-primary">Для домов и участков</h5>
                                <Row>
                                    <Col md={12}>{renderMultiSelect("Тип объекта", "house_types", "houseTypes")}</Col>
                                    <Col md={6}>{renderRangeInput("Площадь участка (сот.)", "plot_area", "plotAreaMin", "plotAreaMax")}</Col>
                                    <Col md={12}>{renderMultiSelect("Отопление", "heating_types", "heatingConditions")}</Col>
                                    <Col md={12}>{renderMultiSelect("Вода", "water_types", "waterConditions")}</Col>
                                    <Col md={12}>{renderMultiSelect("Канализация", "sewage_types", "sewageConditions")}</Col>
                                    <Col md={12}>{renderMultiSelect("Газ", "gaz_types", "gazConditions")}</Col>
                                    <Col md={12}>{renderMultiSelect("Электричество", "electricity_types", "electricityConditions")}</Col>
                                    <Col md={12}>{renderMultiSelect("Крыша", "roof_materials", "roofMaterials")}</Col>
                                </Row>
                            </>
                        )}

                        {filters.category === 'commercial' && (
                            <>
                                <Col md={6}>{renderRangeInput("Площадь общая (м²)", "total_area", "totalAreaMin", "totalAreaMax")}</Col>
                                    
                                <Col md={6}>{renderRangeInput("Этаж", "floor", "floorMin", "floorMax")}</Col>
                                <Col md={6}>{renderRangeInput("Этажность здания", "building_floor", "floorBuildingMin", "floorBuildingMax")}</Col>
                                <hr />
                                <h5 className="mb-3 text-primary">Коммерческая недвижимость</h5>
                                <Row>
                                    {/* Диапазоны (из вашего JSON) */}
                                    
                                    
                                    <Col md={6}>{renderRangeInput("Кол-во помещений", "commercial_rooms", "roomsMin", "roomsMax")}</Col>
                                    
                                   
                                    <Col md={12}>{renderMultiSelect("Расположение", "commercial_locations", "commercialBuildingLocations")}</Col>
                                    
                                    {/* Обычные чекбоксы (если опций мало) или тоже мультиселект */}
                                    <Col md={12}>{renderMultiSelect("Ремонт", "commercial_repairs", "commercialRepairs")}</Col> 
                                    {/* ВНИМАНИЕ: Проверьте тип commercialRepair в IFilterState. Если это массив string[], то ок. */}
                                    
                                    <Col md={12}>{renderMultiSelect("Удобства", "commercial_improvements", "commercialImprovements")}</Col>
                                    {/* Добавьте commercialImprovements в IFilterState и filterUtils.ARRAY_KEYS */}
                                </Row>
                            </>
                        )}
                    </Form>
                </Modal.Body>
                <Modal.Footer className="justify-content-between">
                    <Button variant="link" className="text-danger text-decoration-none" onClick={handleReset}>
                        Сбросить фильтры
                    </Button>
                    <Button variant="success" onClick={() => {
                        setShowModal(false);
                        onSearch(filters);
                    }}>
                        Показать {dynamicOptions?.count || 0} объектов
                    </Button>
                </Modal.Footer>
            </Modal>
        </>
    );
};

export default AdvancedFilterBar;