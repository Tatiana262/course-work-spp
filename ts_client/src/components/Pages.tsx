import React from 'react';
import { Pagination } from 'react-bootstrap';

interface PagesProps {
    totalCount: number;      // Всего объектов
    perPage: number;           // Штук на страницу
    currentPage: number;     // Текущая страница
    onPageChange: (page: number) => void; // Функция смены страницы
}

const Pages: React.FC<PagesProps> = ({ totalCount, perPage, currentPage, onPageChange }) => {
    const pageCount = Math.ceil(totalCount / perPage);
    const siblingCount = 1; // Сколько страниц показывать слева и справа от текущей
    
    // Если страница всего одна, пагинацию не показываем
    if (pageCount <= 1) return null;

     // Логика генерации массива страниц
     const getPaginationRange = () => {
        // Общее количество кнопок, которое мы хотим видеть (first + last + current + 2*siblings + 2*dots)
        // Обычно это около 7 элементов
        const totalPageNumbers = siblingCount * 2 + 5;

        // Случай 1: Если страниц меньше, чем мы хотим показать кнопок - показываем все
        if (totalPageNumbers >= pageCount) {
            return range(1, pageCount);
        }

        const leftSiblingIndex = Math.max(currentPage - siblingCount, 1);
        const rightSiblingIndex = Math.min(currentPage + siblingCount, pageCount);

        // Показываем ли мы точки слева?
        const shouldShowLeftDots = leftSiblingIndex > 2;
        // Показываем ли мы точки справа?
        const shouldShowRightDots = rightSiblingIndex < pageCount - 2;

        const firstPageIndex = 1;
        const lastPageIndex = pageCount;

        // Случай 2: Точки только справа (мы в начале списка)
        if (!shouldShowLeftDots && shouldShowRightDots) {
            let leftItemCount = 3 + 2 * siblingCount;
            let leftRange = range(1, leftItemCount);
            return [...leftRange, 'DOTS', pageCount];
        }

        // Случай 3: Точки только слева (мы в конце списка)
        if (shouldShowLeftDots && !shouldShowRightDots) {
            let rightItemCount = 3 + 2 * siblingCount;
            let rightRange = range(pageCount - rightItemCount + 1, pageCount);
            return [firstPageIndex, 'DOTS', ...rightRange];
        }

        // Случай 4: Точки с обеих сторон (мы в середине)
        if (shouldShowLeftDots && shouldShowRightDots) {
            let middleRange = range(leftSiblingIndex, rightSiblingIndex);
            return [firstPageIndex, 'DOTS', ...middleRange, 'DOTS', lastPageIndex];
        }
        
        return [];
    };

    // Вспомогательная функция для создания массива чисел [start, ..., end]
    const range = (start: number, end: number) => {
        let length = end - start + 1;
        return Array.from({ length }, (_, idx) => idx + start);
    };

    const paginationRange = getPaginationRange();

    return (
        <Pagination className="mt-4 justify-content-center">
            {/* Кнопка "Назад" */}
            <Pagination.Prev 
                onClick={() => onPageChange(currentPage - 1)}
                disabled={currentPage === 1}
            />

            {paginationRange.map((pageNumber, index) => {
                // Если элемент - это маркер 'DOTS', рисуем многоточие
                if (pageNumber === 'DOTS') {
                    return <Pagination.Ellipsis key={`dots-${index}`} disabled />;
                }

                // Иначе рисуем обычную кнопку
                return (
                    <Pagination.Item
                        key={pageNumber}
                        active={pageNumber === currentPage}
                        onClick={() => onPageChange(pageNumber as number)}
                    >
                        {pageNumber}
                    </Pagination.Item>
                );
            })}

            {/* Кнопка "Вперед" */}
            <Pagination.Next 
                onClick={() => onPageChange(currentPage + 1)} 
                disabled={currentPage === pageCount}
            />
        </Pagination>
    );
};

export default Pages;