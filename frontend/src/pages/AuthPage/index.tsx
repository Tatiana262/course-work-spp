import { LoginForm } from '@/features/Auth/ui/LoginForm';
import { RegisterForm } from '@/features/Auth/ui/RegisterForm';
import { useAuth } from '@/features/Auth/model/useAuth';

export const AuthPage = () => {
  // Получаем функции из нашего кастомного хука
  const { login, isLoginLoading, register, isRegisterLoading } = useAuth();

  return (
    <div>
      {/* Передаем функции-мутации в компоненты форм */}
      <LoginForm onSubmit={login} isLoading={isLoginLoading} />
      <hr />
      <RegisterForm onSubmit={register} isLoading={isRegisterLoading} />
    </div>
  );
};

// Вам нужно будет немного доработать LoginForm и RegisterForm,
// чтобы они принимали onSubmit и isLoading как props.