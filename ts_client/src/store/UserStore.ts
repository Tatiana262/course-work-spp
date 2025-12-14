import { makeAutoObservable } from "mobx";

// Описываем, как выглядит объект пользователя, приходящий с бэка
export interface IUser {
    email: string;
    role: string;
    id: number;
    // добавь другие поля если есть
}

export default class UserStore {
    private _isAuth: boolean;
    private _user: IUser | {}; // Пользователь либо пустой объект, либо данные

    constructor() {
        this._isAuth = false;
        this._user = {};
        makeAutoObservable(this);
    }

    setIsAuth(bool: boolean) {
        this._isAuth = bool;
    }

    setUser(user: IUser | {}) {
        this._user = user;
    }

    get isAuth() {
        return this._isAuth;
    }

    get user() {
        return this._user;
    }
}