import { observer } from 'mobx-react-lite';
import { Navigate, useLocation } from 'react-router-dom';
import { userStore } from '@/entities/User/model/userStore';
import { JSX } from 'react';

export const RequireAuth = observer(({ children }: { children: JSX.Element }) => {
  const { isAuthenticated } = userStore;
  const location = useLocation();

  if (!isAuthenticated) {
    // Перенаправляем на страницу логина, но запоминаем, куда пользователь хотел попасть
    return <Navigate to="/auth" state={{ from: location }} replace />;
  }

  return children;
});