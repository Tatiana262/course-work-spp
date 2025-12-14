import { useContext } from "react";
import { Route, Routes, Navigate } from 'react-router-dom';
// Убедись, что routes тоже переведен в TS (код ниже), либо TS будет ругаться на типы
import { authRoutes, publicRoutes } from "../routes"; 
import { MAIN_ROUTE } from "../utils/consts"; // Или MAIN_ROUTE, смотря что у тебя главная
import { Context } from "../main";
import { observer } from "mobx-react-lite"; // <--- ОБЯЗАТЕЛЬНО

const AppRouter = observer(() => {
    const { user } = useContext(Context);

    // console.log("Is Auth:", user.isAuth);

    return (
        <Routes>
            {/* Если авторизован, рендерим закрытые маршруты */}
            {user.isAuth && authRoutes.map(({ path, Component }) => {
                return <Route key={path} path={path} element={<Component />} />
            })}

            {/* Публичные маршруты доступны всем */}
            {publicRoutes.map(({ path, Component }) => {
                return <Route key={path} path={path} element={<Component />} />
            })}

            {/* Если маршрут не найден — редирект на главную */}
            <Route path="*" element={<Navigate to={MAIN_ROUTE} replace />} />
        </Routes>
    );
});

export default AppRouter;