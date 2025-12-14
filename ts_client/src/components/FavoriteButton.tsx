import React, { useState, useEffect, useContext } from 'react';
import { Button, Spinner } from 'react-bootstrap';
import { addToFavorites, removeFromFavorites } from '../http/favoriteAPI';
import { Context } from '../main';
import { observer } from 'mobx-react-lite';

interface Props {
    masterObjectId: string;
    onRemove?: () => void; // Колбэк, чтобы убрать карточку со страницы избранного сразу
}

const FavoriteButton: React.FC<Props> = observer(({ masterObjectId, onRemove }) => {
    const { user, favorites: favoritesIds } = useContext(Context); 
    const [loading, setLoading] = useState(false);

    const isFav = favoritesIds.isFavorite(masterObjectId);

    const handleClick = async (e: React.MouseEvent) => {
        e.stopPropagation(); // Чтобы не переходить на детальную страницу при клике на сердце
        e.preventDefault();

        if (!user.isAuth) {
            alert("Войдите, чтобы добавлять в избранное");
            return;
        }

        setLoading(true);
        try {
            if (isFav) {
                // 1. Запрос на бэк
                await removeFromFavorites(masterObjectId);
                // 2. Обновляем стор
                favoritesIds.removeFavorite(masterObjectId);
                if (onRemove) onRemove();
            } else {
                await addToFavorites(masterObjectId);
                favoritesIds.addFavorite(masterObjectId);
            }
        } catch (error) {
            console.error("Ошибка при работе с избранным", error);
            alert("Не удалось изменить избранное. Авторизуйтесь.");
        } finally {
            setLoading(false);
        }
    };

    return (
        <Button 
            variant={isFav ? "danger" : "outline-secondary"} 
            size="sm"
            onClick={handleClick}
            disabled={loading}
            className="d-flex align-items-center justify-content-center"
            style={{ width: '40px', height: '40px', borderRadius: '50%' }}
        >
            {loading ? (
                <Spinner as="span" animation="border" size="sm" />
            ) : (
                <i className={`bi ${isFav ? 'bi-heart-fill' : 'bi-heart'}`} style={{ fontSize: '1.2rem' }}></i>
            )}
        </Button>
    );
});

export default FavoriteButton;