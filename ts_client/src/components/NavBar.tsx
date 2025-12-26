import { useContext } from "react";
import { Container, Nav, Navbar, Button} from 'react-bootstrap';
import { Context } from "../main";
import { NavLink } from "react-router-dom"
import { ADMIN_ROUTE, FAVORITES_ROUTE, LOGIN_ROUTE, MAIN_ROUTE } from "../utils/consts"; // Добавь SHOP_ROUTE если есть
import { observer } from "mobx-react-lite"
import { useNavigate } from "react-router-dom";
import type { IUser } from "../store/UserStore";

const NavBar = observer(() => {
    const { user } = useContext(Context);
    const navigate = useNavigate();

    const logOut = () => {
        user.setUser({});
        user.setIsAuth(false);
        localStorage.removeItem('token');
        // navigate(MAIN_ROUTE); // Раскомментируй, если есть константа
        // navigate(LOGIN_ROUTE);
    }

    // Приведение типа: мы проверяем isAuth, значит user.user точно содержит данные
    // Но лучше сделать Type Guard в сторе. Пока сделаем простое приведение для удобства.
    const userData = user.user as IUser; 

    return (
        <Navbar bg="dark" data-bs-theme="dark">
            <Container>
                <NavLink style={{ color: 'white' }} to={MAIN_ROUTE}>RealEstate</NavLink>
                {user.isAuth ?
                    <Nav className="ml-auto">
                        {/* userData может быть пустым объектом теоретически, но isAuth=true защищает нас */}
                        {userData.role === 'admin' &&
                            <Button
                                variant={"outline-light"}
                                className="mx-2"
                                onClick={() => navigate(ADMIN_ROUTE)}
                            >
                                Админ панель
                            </Button>
                        }
                        <Button
                            variant={"outline-light"}
                            onClick={() => logOut()}
                        >
                            Выйти
                        </Button>
                        <Button 
                            variant={"outline-danger"} 
                            className="ms-2"
                            onClick={() => navigate(FAVORITES_ROUTE)}
                        >
                            <i className="bi bi-heart-fill me-1"></i> Избранное
                        </Button>
                    </Nav> :
                    <Nav className="ml-auto">
                        <Button variant={"outline-light"} onClick={() => navigate(LOGIN_ROUTE)}>Авторизация</Button>
                    </Nav>
                }
            </Container>
        </Navbar>
    );
});

export default NavBar;