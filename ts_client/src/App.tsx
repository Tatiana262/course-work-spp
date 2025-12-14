import { useContext, useEffect, useState } from "react";
import { BrowserRouter } from "react-router-dom";
import NavBar from "./components/NavBar";
import { observer } from "mobx-react-lite";
import { Context } from "./main";
import { validate } from "./http/userAPI";
import { Spinner } from "react-bootstrap";
import AppRouter from "./components/AppRouter";
import { fetchFavoriteIDs } from "./http/favoriteAPI";
import { subscribeToTasks } from "./http/adminAPI";
// import { AxiosError } from "axios"; // Тип для ошибки

const App = observer(() => {
  const { user, favorites, actualization } = useContext(Context);
  const [loading, setLoading] = useState<boolean>(true); // Явно указываем тип

  useEffect(() => {
    validate()
      .then(data => {
        // data теперь имеет тип IUser, TS не будет ругаться
        user.setUser(data);
        user.setIsAuth(true);

        fetchFavoriteIDs().then(ids => {
          console.log(ids)
          favorites.setFavorites(ids);
        }).catch(e => console.error("Не удалось загрузить избранное", e));

      })
      .catch(() => {
          // Обрабатываем ошибку безопасно
          // console.log(err.message);
          user.setIsAuth(false);
          localStorage.removeItem('token');
      })
      .finally(() => setLoading(false));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (user.isAuth) {
        // Подписываемся на события задач ГЛОБАЛЬНО
        const unsubscribe = subscribeToTasks(
            (task) => {
                // Фильтруем только задачи актуализации по ID
                if (task.type === 'ACTUALIZE_BY_ID' && task.status === 'completed') {
                    
                    // Пытаемся достать ID объекта. 
                    //  Если он есть в task.details (лучше добавить на бэке)
                    // const objectId = task.details?.object_id;
                    const objectId = task.result_summary?.id

                    if (objectId) {
                      actualization.remove(objectId);
                      actualization.markAsUpdated(objectId); 
                          
                      console.log(`Объект ${objectId} обновлен!`);
                    }                    
                }
            },
            (err) => console.error("SSE Error", err)
        );

        return () => unsubscribe();
    }
  }, [user.isAuth]);

  if (loading) {
    return <Spinner animation="grow" />;
  }

  return (
    <BrowserRouter>
      <NavBar />
      <AppRouter />
    </BrowserRouter>
  );
});

export default App;