import React, { useEffect, useState } from 'react';
import { Container, Spinner, Alert } from 'react-bootstrap';
import { useSearchParams } from 'react-router-dom';
import REObjectItem from '../components/RealEstateObjectItem';
import AdvancedFilterBar from '../components/AdvancedFilterBar'; // <-- Обнови импорт
import Pages from '../components/Pages';
import { fetchObjects } from '../http/objectsAPI';
import type { IObjectCardResponse } from '../types/realEstateObjects';
import type { IFilterState } from '../types/filter';
import { ARRAY_KEYS } from '../utils/filterUtils'; // <-- Импортируем константу

const Listings = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const [properties, setProperties] = useState<IObjectCardResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [totalCount, setTotalCount] = useState(0);

    const LIMIT = 5;
    const currentPage = parseInt(searchParams.get('page') || '1');

    const loadProperties = () => {
        setLoading(true);
        
        // 1. ПРАВИЛЬНЫЙ ПАРСИНГ URL ДЛЯ API
        const params: any = {};
        
        // Перебираем все ключи, которые есть в URL
        searchParams.forEach((value, key) => {
            // Если ключ входит в список массивов (например, rooms)
            // @ts-ignore
            if (ARRAY_KEYS.includes(key)) {
                // Если массив еще не создан, создаем
                if (!params[key]) {
                    // Используем getAll, чтобы получить ВСЕ значения ['1', '2']
                    params[key] = searchParams.getAll(key);
                }
            } else {
                // Обычные поля (category, priceMin)
                params[key] = value;
            }
        });

        // Пагинация
        params.page = currentPage;
        params.perPage = LIMIT;

        // console.log("Отправляем на сервер:", params); // Для отладки

        fetchObjects(params)
            .then(data => {
                setProperties(data.objects);
                setTotalCount(data.total);
                window.scrollTo(0, 0);
            })
            .catch(err => console.error(err))
            .finally(() => setLoading(() => false)); // Колбэк чтобы точно обновить
    };

    useEffect(() => {
        loadProperties();
    }, [searchParams]);

    // Обработчик поиска из FilterBar
    const handleSearch = (filters: IFilterState) => {
        // Превращаем объект фильтров в URLSearchParams-совместимый объект
        const params: any = {};
        
        Object.entries(filters).forEach(([key, value]) => {
            // Пропускаем пустые значения
            if (value === '' || value === null || (Array.isArray(value) && value.length === 0)) {
                return;
            }
            params[key] = value;
        });
        
        // Сбрасываем на 1 страницу при новом поиске
        params.page = '1'; 

        setSearchParams(params);
    };

    const handlePageChange = (page: number) => {
        // При смене страницы нужно сохранить текущие фильтры!
        // setSearchParams сам смерджит старые и новые параметры, 
        // но лучше явно передать текущий searchParams + новую страницу
        const current = new URLSearchParams(searchParams);
        current.set('page', page.toString());
        setSearchParams(current);
    };

    return (
        <Container className="mt-4">
            <AdvancedFilterBar onSearch={handleSearch} />

            <div className="mb-3 text-muted">
                Найдено объявлений: <b>{totalCount}</b>
            </div>

            {loading ? (
                <div className="d-flex justify-content-center mt-5">
                    <Spinner animation="border" variant="primary" />
                </div>
            ) : properties && properties.length > 0 ? (
                <div>
                    {properties.map(property => (
                        <REObjectItem key={property.id} isFavoritePage={false} property={property} />
                    ))}
                    <Pages 
                        totalCount={totalCount} 
                        perPage={LIMIT} 
                        currentPage={currentPage}
                        onPageChange={handlePageChange}
                    />
                </div>
            ) : (
                <Alert variant="info">
                    По вашему запросу ничего не найдено.
                </Alert>
            )}
        </Container>
    );
};

export default Listings;