import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { loginUser, registerUser } from '../api/authApi';
import { userStore } from '@/entities/User/model/userStore';

export const useAuth = () => {
  const navigate = useNavigate();
  // В MobX сторе вы, вероятно, используете его напрямую, а не через `setUser`
  // const { setUser } = userStore;
  // Давайте предположим, что вы вызываете action `userStore.setUser(...)`

  // --- Мутация для логина ---
  const { mutate: login, isPending: isLoginPending } = useMutation({ // <-- ИЗМЕНЕНИЕ: isLoading -> isPending
    mutationFn: loginUser,
    onSuccess: (data) => {
      localStorage.setItem('authToken', data.token);

      // Обновляем MobX стор
      userStore.setUser({
        id: data.user_id,
        email: data.email,
        role: data.role,
      });

      navigate('/');
    },
    onError: (error) => {
      console.error('Login failed:', error);
      // Здесь можно добавить логику для показа уведомлений об ошибке
      // например, с помощью библиотеки react-toastify
    },
  });

  // --- Мутация для регистрации ---
  const { mutate: register, isPending: isRegisterPending } = useMutation({ // <-- ИЗМЕНЕНИЕ: isLoading -> isPending
    mutationFn: registerUser,
    onSuccess: (data) => {
      localStorage.setItem('authToken', data.token);
      userStore.setUser({
        id: data.user_id,
        email: data.email,
        role: data.role,
      });
      navigate('/');
    },
    onError: (error) => {
      console.error('Registration failed:', error);
    },
  });

  // Хук возвращает новые имена свойств
  return {
    login,
    isLoginLoading: isLoginPending, // <-- Возвращаем под старым именем для удобства
    register,
    isRegisterLoading: isRegisterPending, // <-- или используем новые имена везде
  };
};