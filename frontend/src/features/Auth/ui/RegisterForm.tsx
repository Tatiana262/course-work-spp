import { useForm, SubmitHandler } from 'react-hook-form';
import { RegisterRequestDto } from '../api/authApi'; // Импортируем типы из API

// Определяем props, которые будет принимать компонент
interface RegisterFormProps {
  onSubmit: (data: RegisterRequestDto) => void;
  isLoading: boolean;
}

export const RegisterForm = ({ onSubmit, isLoading }: RegisterFormProps) => {
  const { 
    register, 
    handleSubmit, 
    formState: { errors } // Достаем состояние ошибок
  } = useForm<RegisterRequestDto>(); // Типизируем форму

  // `handleSubmit` из react-hook-form вызывает наш `onSubmit` только если валидация прошла
  const onFormSubmit: SubmitHandler<RegisterRequestDto> = (data) => {
    onSubmit(data);
  };

  return (
    // Передаем наш обработчик в handleSubmit
    <form onSubmit={handleSubmit(onFormSubmit)}>
      <h2>Регистрация</h2>
      <div>
        <input 
          {...register('email', { required: 'Email обязателен' })} 
          placeholder="Email" 
          type="email" 
          disabled={isLoading} // Блокируем поля во время загрузки
        />
        {/* Отображаем ошибку, если она есть */}
        {errors.email && <p style={{ color: 'red' }}>{errors.email.message}</p>}
      </div>
      <div>
        <input 
          {...register('password', { required: 'Пароль обязателен' })} 
          placeholder="Пароль" 
          type="password"
          disabled={isLoading}
        />
        {errors.password && <p style={{ color: 'red' }}>{errors.password.message}</p>}
      </div>
      {/* Блокируем кнопку во время отправки и показываем текст "Загрузка..." */}
      <button type="submit" disabled={isLoading}>
        {isLoading ? 'Регистрация...' : 'Зарегистрироваться'}
      </button>
    </form>
  );
};