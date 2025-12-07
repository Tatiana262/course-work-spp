import { observer } from 'mobx-react-lite';
import { userStore } from '@/entities/User/model/userStore';
import { Link } from 'react-router-dom';

export const Header = observer(() => {
  const { isAuthenticated, user, setUser } = userStore;

  const handleLogout = () => {
    setUser(null);
    localStorage.removeItem('authToken');
  };

  return (
    <header>
      <nav>
        <Link to="/">Главная</Link> | <Link to="/favorites">Избранное</Link>
        {isAuthenticated ? (
          <>
            <span> | Привет, {user?.id}!</span>
            <button onClick={handleLogout}>Выйти</button>
          </>
        ) : (
          <Link to="/auth"> | Войти</Link>
        )}
      </nav>
    </header>
  );
});