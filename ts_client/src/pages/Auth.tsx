import React, { useContext, useState } from 'react';
import { Button, Card, Container, Form } from "react-bootstrap";
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LOGIN_ROUTE, REGISTRATION_ROUTE, MAIN_ROUTE } from '../utils/consts'
import classes from '../pagesStyles/AuthStyle.module.css'
import { registration, login } from '../http/userAPI';
import { observer } from 'mobx-react-lite';
import { Context } from '../main';
import { AxiosError } from 'axios';

const Auth = observer(() => {
    const { user } = useContext(Context);
    const location = useLocation();
    const navigate = useNavigate();
    const isLogin = location.pathname === LOGIN_ROUTE;
    const [email, setEmail] = useState<string>('');
    const [password, setPassword] = useState<string>('');

    const click = async () => {
        try {
            let data;
            if (isLogin) {
                data = await login(email, password);
            } else {
                data = await registration(email, password);
            }
            // Здесь data - это IUser, метод setUser ожидает IUser. Всё сходится.
            user.setUser(data);
            user.setIsAuth(true);
            navigate(MAIN_ROUTE);
        }
        catch (e) {
            // Приводим ошибку к типу AxiosError, чтобы достать response
            const err = e as AxiosError<{message: string}>;
            alert(err.response?.data?.message || "Произошла ошибка");
        }
    }

    return (
        <Container
            className='d-flex justify-content-center align-items-center'
            style={{ height: window.innerHeight - 54 }}
        >
            <Card style={{ width: '600px' }} className='p-5'>
                <h2 className="m-auto">{isLogin ? 'Авторизация' : 'Регистрация'}</h2>
                <Form className='d-flex flex-column'>
                    <Form.Control
                        type="email"
                        className='mt-3'
                        placeholder="Введите email"
                        value={email}
                        // Типизация события изменения инпута
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setEmail(e.target.value)}
                    />
                    <Form.Control
                        type="password"
                        className='mt-3'
                        placeholder="Введите пароль"
                        value={password}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
                    />
                    <div className={classes.wrapper}>
                        {isLogin ?
                            <div>
                                Нет аккаунта? <NavLink to={REGISTRATION_ROUTE}>Зарегистрироваться</NavLink>
                            </div> :
                            <div>
                                Есть аккаунт? <NavLink to={LOGIN_ROUTE}>Войти</NavLink>
                            </div>
                        }
                        <Button
                            variant={"outline-success"}
                            onClick={click}
                        >
                            {isLogin ? 'Войти' : 'Регистрация'}
                        </Button>
                    </div>
                </Form>
            </Card>
        </Container>
    );
});

export default Auth;