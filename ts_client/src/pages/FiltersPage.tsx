import { useState } from 'react';
import { Container } from 'react-bootstrap';
import type { IFilterState } from '../types/filter';
import { useNavigate, createSearchParams } from 'react-router-dom';
import { LISTINGS_ROUTE } from '../utils/consts';
import AdvancedFilterBar from '../components/AdvancedFilterBar';

const Filters = () => {
    const navigate = useNavigate();

    const handleSearch = (filters: IFilterState) => {
        // Используем any, чтобы можно было класть массивы строк
        const params: any = {};
        
        Object.entries(filters).forEach(([key, value]) => {
            // Фильтруем пустые значения
            if (value === '' || value === null || (Array.isArray(value) && value.length === 0)) {
                return;
            }
            if (Array.isArray(value)) {
                // Склеиваем массив в строку через запятую
                params[key] = value.join(',');
            } else {
                params[key] = value;
            }
        });
        
        navigate({
            pathname: LISTINGS_ROUTE,
            search: createSearchParams(params).toString()
        });
    };

    return (
        <Container className="mt-4">
            <h1 className="text-center mb-5">Агрегатор недвижимости Беларуси</h1>
            
            {/* На главной FilterBar просто перекидывает на страницу результатов */}
            <AdvancedFilterBar onSearch={handleSearch} />
            
            {/* Тут можно добавить "Популярные предложения" или промо-блоки */}
        </Container>
    );
};

export default Filters;