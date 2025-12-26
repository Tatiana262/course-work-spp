import React, { useEffect, useState, useContext } from 'react';
import { Container, Spinner, Alert } from 'react-bootstrap';
import REObjectItem from '../components/RealEstateObjectItem';
import { fetchFavorites } from '../http/favoriteAPI';
import type { IObjectCardResponse } from '../types/realEstateObjects';
import { Context } from '../main'; // Чтобы проверить авторизацию
import { observer } from 'mobx-react-lite';
import Pages from '../components/Pages';

const Favorites = observer(() => {
    const { user, actualization } = useContext(Context);
    const [favorites, setFavorites] = useState<IObjectCardResponse[]>([]);
    const [loading, setLoading] = useState(true);
    const [totalCount, setTotalCount] = useState(0);
    const [page, setPage] = useState(1);
    const limit = 5;

    // Функция загрузки (вынесли, чтобы вызывать при удалении)
    const loadFavorites = (isSilent = false) => {
        if (!isSilent) setLoading(true);

        fetchFavorites(page, limit)
            .then(responseData => {
                console.log(responseData.data)
                setFavorites(responseData.data || []); 
                setTotalCount(responseData.total || 0);
            })
            .catch(err => console.error("Ошибка загрузки избранного", err))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        if (user.isAuth) {
            loadFavorites();
        } else {
            setLoading(false);
        }
    }, [user.isAuth, page]);

    
    favorites.forEach((fav) => {
        console.log(fav.master_object_id)
    })
 
    actualization.updates.forEach((val, key)=>{
        console.log(key)
    })
   

    useEffect(() => {
        // Этот эффект сработает, когда в сторе изменится version (кто-то обновился)
        // проверка, касается ли это обновление текущего списка
        
        const hasRelevantUpdates = favorites.some(fav => 
            actualization.updates.has(fav.master_object_id)
        );

        if (hasRelevantUpdates) {
            console.log("Найдены обновления для текущего списка! Перезагружаем...");
            loadFavorites(true); // Тихая перезагрузка
        }
        
    }, [actualization.version]); // Зависимость от версии стора

    if (!user.isAuth) {
        return (
            <Container className="mt-5">
                <Alert variant="warning">
                    Пожалуйста, авторизуйтесь, чтобы просматривать избранное.
                </Alert>
            </Container>
        );
    }

    // Если удалили объект, обновляем список
    const handleRemoveItem = (deletedId: string) => {
        // Оптимистичное обновление: сразу убираем из UI
        setFavorites(prev => prev.filter(item => item.master_object_id !== deletedId));
        // Можно перезапросить список, чтобы обновился totalCount и пагинация
        // loadFavorites(); 
    };

    return (
        <Container className="mt-4">
            <h2 className="mb-4">Моё избранное ({totalCount})</h2>

            {loading ? (
                <div className="d-flex justify-content-center mt-5">
                    <Spinner animation="border" variant="danger" />
                </div>
            ) : favorites.length > 0 ? (
                <div>
                    {favorites.map(property => (
                        <REObjectItem 
                            key={property.id} 
                            property={property} 
                            isFavoritePage={true}
                            onRemoveFromFav={() => handleRemoveItem(property.master_object_id)}
                        />
                    ))}
                    
                    <Pages 
                        totalCount={totalCount} 
                        perPage={limit} 
                        currentPage={page} 
                        onPageChange={setPage} 
                    />
                </div>
            ) : (
                <Alert variant="info">
                    Список избранного пуст. Добавляйте объекты, нажимая на сердечко.
                </Alert>
            )}
        </Container>
    );
});

export default Favorites;