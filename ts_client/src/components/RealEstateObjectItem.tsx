import React from 'react';
import { Card, Badge, Button } from 'react-bootstrap';
import type { IObjectCardResponse } from '../types/realEstateObjects';
import { useNavigate } from 'react-router-dom';
import FavoriteButton from './FavoriteButton';
import ActualizeButton from './ActualizeButton';

interface PropertyItemProps {
    property: IObjectCardResponse;
    isFavoritePage: boolean;
    onRemoveFromFav?: () => void; // Функция обновления списка
}

const REObjectItem: React.FC<PropertyItemProps> = ({ property, isFavoritePage, onRemoveFromFav }) => {
    const navigate = useNavigate();

    // Логика для архивных объектов
    const isArchived = property.status === 'archived';

    // Стили для архива
    const cardStyle: React.CSSProperties = {
        overflow: 'hidden',
        transition: '0.2s',
        // Если архив - делаем серым и полупрозрачным
        opacity: isArchived ? 0.7 : 1,
        filter: isArchived ? 'grayscale(100%)' : 'none',
        // запретить клик по архивному:
        // pointerEvents: isArchived ? 'none' : 'auto', 
    };

    // Заглушка, если картинки нет
    const imageUrl = property.images && property.images.length > 0 
        ? property.images[0] 
        : 'https://via.placeholder.com/300x200?text=No+Image';

    return (
        <Card className="mb-3 shadow-sm position-relative" style={cardStyle}>
             {/* Бейдж Архива */}
            {isArchived && (
                <Badge 
                    bg="secondary" 
                    className="position-absolute top-0 start-0 m-3 p-2" 
                    style={{ zIndex: 10 }}
                >
                    Снято с публикации
                </Badge>
            )}

            <div className="d-flex flex-column flex-md-row">
                <div 
                    style={{ width: '100%', maxWidth: '300px', height: '200px', flexShrink: 0, cursor: 'pointer' }}
                    onClick={() => !isArchived && navigate(`/objects/${property.id}`)} // Блокируем переход если архив (по вашему желанию)
                >
                    <Card.Img 
                        src={imageUrl} 
                        style={{ width: '100%', height: '100%', objectFit: 'cover' }} 
                    />
                </div>

                <div className="d-flex flex-column flex-grow-1 p-3">
                    <div className="d-flex justify-content-between align-items-start">
                        <div style={{ maxWidth: '85%' }}>
                            <h5 className="mb-1 text-truncate" title={property.title}>{property.title}</h5>
                            <small className="text-muted">{property.address}</small>
                        </div>
                        
                        {/* Кнопка Лайка */}
                        <div style={{ pointerEvents: 'auto' }}> {/* Возвращаем кликабельность для кнопки */}
                            <FavoriteButton 
                                masterObjectId={property.master_object_id}
                                onRemove={onRemoveFromFav}
                            />
                        </div>
                    </div>

                    <div className="mt-2">
                        <h4 className="text-primary">
                            {property.price_byn > 0 ? (
                                <>
                                    {/* Если цена есть */}
                                    {property.price_byn.toLocaleString('ru-RU')} BYN
                                    <span className="text-muted fs-6 ms-2">
                                        (~{property.price_usd?.toLocaleString('ru-RU')} $)
                                    </span>
                                </>
                            ) : (
                                /* Если цены нет (0) */
                                "Договорная"
                            )}
                        </h4>
                    </div>

                    <div className="mt-auto d-flex justify-content-between align-items-center">
                        <div className="text-muted small">
                            {property.category} | {property.deal_type === 'sale' ? 'Продажа' : 'Аренда'}
                        </div>
                        
                        <div className="d-flex gap-2">
                            {/* Показываем кнопку актуализации только в избранном */}
                            {isFavoritePage && (
                                <ActualizeButton 
                                    master_object_id={property.master_object_id} 
                                    variant="outline-secondary" // Серый стиль для списка
                                />
                            )}
                            
                            <Button 
                                variant="outline-primary" 
                                onClick={() => navigate(`/objects/${property.id}`)}
                            >
                                Подробнее
                            </Button>
                        </div>        
                    </div>
                </div>
            </div>
        </Card>
    );
};

export default REObjectItem;


// {!isArchived ? (
                           
//     ) : (
//         <Button variant="secondary" disabled>Неактивно</Button>
//     )}